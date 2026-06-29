/* ZlefRemote panel plugin — UI + agent control (GTK3 only).
 *
 * Spawns the `zlefremote-agent` binary in `-machine` mode, reads its
 * line-oriented "@zr key=value" protocol from stdout, and reflects pairing
 * state (URL + QR code) in a small popup. No libxfce4panel dependency, so the
 * same code drives both the panel plugin and the standalone test window.
 */
#include "zlefremote.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <sys/types.h>

/* ── tiny i18n ────────────────────────────────────────────────────────────
 * Two locales (en default + fr) → resolve silently from the environment, no
 * language picker (per the zlef DA i18n rule for 2-locale projects). */

typedef struct { const char *key, *en, *fr; } ZrStr;

static const ZrStr STRINGS[] = {
  { "title",        "ZlefRemote",                       "ZlefRemote" },
  { "tagline",      "Your phone is the trackpad",        "Votre téléphone devient le trackpad" },
  { "mode_lan",     "Local network",                     "Réseau local" },
  { "mode_lan_d",   "Same Wi-Fi · fastest, fully local", "Même Wi-Fi · le plus rapide, 100% local" },
  { "mode_remote",  "Remote",                            "À distance" },
  { "mode_remote_d","From anywhere · end-to-end encrypted","Depuis partout · chiffré de bout en bout" },
  { "start",        "Start",                             "Démarrer" },
  { "stop",         "Stop",                              "Arrêter" },
  { "st_idle",      "Stopped",                           "Arrêté" },
  { "st_starting",  "Starting…",                         "Démarrage…" },
  { "st_waiting",   "Scan the code with your phone",      "Scannez le code avec votre téléphone" },
  { "st_paired",    "Phone connected",                   "Téléphone connecté" },
  { "remember",     "Remember this computer",            "Mémoriser cet ordinateur" },
  { "remember_d",   "Saved phones reconnect in one tap",  "Les téléphones enregistrés se reconnectent en un geste" },
  { "remember_old", "Update the ZlefRemote agent (1.1.0+) to use this",
                    "Mettez à jour l'agent ZlefRemote (1.1.0+) pour l'utiliser" },
  { "saved_hint",   "Saved — your phone can reconnect to this computer anytime",
                    "Enregistré — votre téléphone peut se reconnecter à cet ordinateur à tout moment" },
  { "clients_1",    "1 phone connected",                 "1 téléphone connecté" },
  { "clients_n",    "%d phones connected",               "%d téléphones connectés" },
  { "copy",         "Copy link",                         "Copier le lien" },
  { "copied",       "Copied!",                           "Copié !" },
  { "open_phone",   "Or open this link on your phone:",   "Ou ouvrez ce lien sur votre téléphone :" },
  { "no_agent",     "Agent not found. Install the ZlefRemote agent.",
                    "Agent introuvable. Installez l'agent ZlefRemote." },
  { NULL, NULL, NULL }
};

static gboolean lang_is_fr(void) {
  const char *l = g_getenv("LC_ALL");
  if (!l || !*l) l = g_getenv("LC_MESSAGES");
  if (!l || !*l) l = g_getenv("LANG");
  return l && (g_ascii_strncasecmp(l, "fr", 2) == 0);
}

const char *zr_t(const char *key) {
  gboolean fr = lang_is_fr();
  for (const ZrStr *s = STRINGS; s->key; s++)
    if (strcmp(s->key, key) == 0)
      return fr ? s->fr : s->en;
  return key;
}

/* ── app state ──────────────────────────────────────────────────────────── */

struct _ZrApp {
  /* agent process */
  GPid       pid;          /* 0 when not running */
  guint      child_watch;
  GIOChannel *out;
  guint      out_watch;
  GString   *linebuf;      /* partial stdout line accumulator */

  gboolean   remote_mode;  /* TRUE = remote, FALSE = lan */
  gboolean   persistent;   /* TRUE once the agent reports a remembered identity */
  gboolean   agent_remember; /* TRUE if the resolved agent supports --remember */
  ZrStatus   status;
  char      *url;
  char      *qr_path;
  char      *agent_path;   /* resolved agent binary, or NULL */
  GHashTable *peers;       /* connected clients: id(int) -> ip(char*)  */

  /* widgets */
  GtkWidget *root;
  GtkWidget *mode_lan;
  GtkWidget *mode_remote;
  GtkWidget *remember_chk; /* "remember this computer" (remote mode only) */
  GtkWidget *saved_lbl;    /* shown in the pairing box when persistent */
  GtkWidget *start_btn;
  GtkWidget *start_lbl;
  GtkWidget *status_lbl;
  GtkWidget *clients_lbl;  /* "N connected" + IP list, shown when count > 0 */
  GtkWidget *pair_box;     /* QR + url, shown only when waiting/paired */
  GtkWidget *qr_img;
  GtkWidget *url_entry;
  GtkWidget *copy_btn;
  GtkWidget *copy_lbl;

  void     (*status_cb)(ZrStatus, gpointer);
  gpointer   status_user;
};

/* ── agent binary discovery ───────────────────────────────────────────────
 * Search order: $ZLEFREMOTE_AGENT, $PATH, then the bundled install locations. */

static char *find_agent(void) {
  const char *env = g_getenv("ZLEFREMOTE_AGENT");
  if (env && *env && g_file_test(env, G_FILE_TEST_IS_EXECUTABLE))
    return g_strdup(env);

  char *p = g_find_program_in_path("zlefremote-agent");
  if (p) return p;

  const char *home = g_get_home_dir();
  char *candidates[] = {
    g_build_filename(g_get_user_data_dir(), "zlefremote", "zlefremote-agent", NULL),
    g_build_filename(home, ".local", "share", "zlefremote", "zlefremote-agent", NULL),
    g_strdup("/usr/local/lib/zlefremote/zlefremote-agent"),
    g_strdup("/usr/lib/zlefremote/zlefremote-agent"),
    NULL
  };
  char *found = NULL;
  for (int i = 0; candidates[i]; i++) {
    if (!found && g_file_test(candidates[i], G_FILE_TEST_IS_EXECUTABLE))
      found = g_strdup(candidates[i]);
    g_free(candidates[i]);
  }
  return found;
}

/* Does the resolved agent understand --remember? Older agents (<1.1.0) abort on
 * the unknown flag and exit immediately, which looked like "Start does nothing".
 * We probe `agent -help` once and only offer Remember when it's actually there,
 * so the option degrades gracefully across agent versions on $PATH. */
static gboolean probe_remember(const char *agent) {
  if (!agent) return FALSE;
  char *out = NULL, *err = NULL;
  char *argv[] = { (char*)agent, (char*)"-help", NULL };
  /* flags = 0: capture both stdout and stderr (Go's flag prints usage to stderr) */
  gboolean spawned = g_spawn_sync(NULL, argv, NULL, 0,
      NULL, NULL, &out, &err, NULL, NULL);
  gboolean has = (out && strstr(out, "-remember")) || (err && strstr(err, "-remember"));
  g_free(out); g_free(err);
  (void) spawned;
  return has;
}

/* ── status transitions ───────────────────────────────────────────────────*/

static void set_status(ZrApp *app, ZrStatus st) {
  app->status = st;

  const char *txt = "";
  switch (st) {
    case ZR_IDLE:     txt = zr_t("st_idle");     break;
    case ZR_STARTING: txt = zr_t("st_starting"); break;
    case ZR_WAITING:  txt = zr_t("st_waiting");  break;
    case ZR_PAIRED:   txt = zr_t("st_paired");   break;
  }
  if (app->status_lbl)
    gtk_label_set_text(GTK_LABEL(app->status_lbl), txt);

  gboolean running = (st != ZR_IDLE);
  gtk_label_set_text(GTK_LABEL(app->start_lbl), running ? zr_t("stop") : zr_t("start"));
  gtk_widget_set_sensitive(app->mode_lan, !running);
  gtk_widget_set_sensitive(app->mode_remote, !running);
  /* "remember" only applies to remote mode, needs an agent that supports it,
   * and can't be toggled mid-session */
  if (app->remember_chk)
    gtk_widget_set_sensitive(app->remember_chk,
        !running && app->remote_mode && app->agent_remember);

  gboolean show_pair = (st == ZR_WAITING || st == ZR_PAIRED) && app->url;
  gtk_widget_set_visible(app->pair_box, show_pair);
  if (app->saved_lbl)
    gtk_widget_set_visible(app->saved_lbl, show_pair && app->persistent);

  /* paired: dim the QR slightly via opacity to signal "in use" */
  if (app->qr_img)
    gtk_widget_set_opacity(app->qr_img, st == ZR_PAIRED ? 0.4 : 1.0);

  if (app->status_cb)
    app->status_cb(st, app->status_user);
}

/* ── connected-clients roster ─────────────────────────────────────────────*/

/* Re-render the "N connected" label + the IP list from the peers table. The
 * label is hidden when nobody is connected. */
static void render_clients(ZrApp *app) {
  if (!app->clients_lbl) return;
  guint n = app->peers ? g_hash_table_size(app->peers) : 0;
  if (n == 0) {
    gtk_widget_set_visible(app->clients_lbl, FALSE);
    return;
  }

  /* headline: "N phones connected" */
  char *head = (n == 1)
    ? g_strdup(zr_t("clients_1"))
    : g_strdup_printf(zr_t("clients_n"), n);

  /* one IP per line under the headline (or "unknown" when not reported) */
  GString *body = g_string_new(NULL);
  GHashTableIter it;
  gpointer k, v;
  g_hash_table_iter_init(&it, app->peers);
  while (g_hash_table_iter_next(&it, &k, &v)) {
    const char *ip = v && *(char*)v ? (char*)v : "unknown";
    g_string_append_c(body, '\n');
    g_string_append(body, ip);
  }

  char *markup = g_markup_printf_escaped(
      "<b>%s</b><span size='small'>%s</span>", head, body->str);
  gtk_label_set_markup(GTK_LABEL(app->clients_lbl), markup);
  gtk_widget_set_visible(app->clients_lbl, TRUE);

  g_free(markup);
  g_string_free(body, TRUE);
  g_free(head);
}

static void peers_clear(ZrApp *app) {
  if (app->peers) g_hash_table_remove_all(app->peers);
  render_clients(app);
}

/* ── stdout protocol parsing ──────────────────────────────────────────────*/

static void handle_line(ZrApp *app, const char *line) {
  if (g_str_has_prefix(line, "@zr ")) {
    const char *kv = line + 4;
    const char *eq = strchr(kv, '=');
    if (!eq) return;
    char *key = g_strndup(kv, eq - kv);
    const char *val = eq + 1;

    if (strcmp(key, "url") == 0) {
      g_free(app->url); app->url = g_strdup(val);
      if (app->url_entry)
        gtk_entry_set_text(GTK_ENTRY(app->url_entry), app->url);
    } else if (strcmp(key, "qr") == 0) {
      g_free(app->qr_path); app->qr_path = g_strdup(val);
      GError *e = NULL;
      GdkPixbuf *pb = gdk_pixbuf_new_from_file_at_scale(app->qr_path, 200, 200, TRUE, &e);
      if (pb) {
        gtk_image_set_from_pixbuf(GTK_IMAGE(app->qr_img), pb);
        g_object_unref(pb);
      } else if (e) {
        g_clear_error(&e);
      }
    } else if (strcmp(key, "status") == 0 && strcmp(val, "waiting") == 0) {
      set_status(app, ZR_WAITING);
    } else if (strcmp(key, "peer") == 0) {
      /* "join <id> <ip>" | "leave <id>" */
      if (g_str_has_prefix(val, "join ")) {
        int id = 0; char ip[256] = "";
        if (sscanf(val + 5, "%d %255s", &id, ip) >= 1)
          g_hash_table_insert(app->peers, GINT_TO_POINTER(id), g_strdup(ip));
        render_clients(app);
      } else if (g_str_has_prefix(val, "leave ")) {
        int id = atoi(val + 6);
        g_hash_table_remove(app->peers, GINT_TO_POINTER(id));
        render_clients(app);
      }
    } else if (strcmp(key, "clients") == 0) {
      /* authoritative count; 0 means the roster was reset (relay drop) */
      if (atoi(val) == 0) peers_clear(app);
    } else if (strcmp(key, "persistent") == 0) {
      /* the agent has a remembered identity → its room is stable and the phone
       * can save it; surface the "saved" hint in the pairing box. */
      app->persistent = (strcmp(val, "1") == 0);
      if (app->saved_lbl)
        gtk_widget_set_visible(app->saved_lbl,
            app->persistent && (app->status == ZR_WAITING || app->status == ZR_PAIRED));
    } else if (strcmp(key, "event") == 0) {
      if (strcmp(val, "paired") == 0)      set_status(app, ZR_PAIRED);
      else if (strcmp(val, "disconnect") == 0) { peers_clear(app); if (app->pid) set_status(app, ZR_WAITING); }
    }
    g_free(key);
  }
}

static gboolean on_agent_output(GIOChannel *src, GIOCondition cond, gpointer data) {
  ZrApp *app = data;
  gchar buf[4096];
  gsize n = 0;
  GError *e = NULL;
  GIOStatus s;

  while ((s = g_io_channel_read_chars(src, buf, sizeof buf, &n, &e)) == G_IO_STATUS_NORMAL && n > 0) {
    g_string_append_len(app->linebuf, buf, n);
    char *nl;
    while ((nl = strchr(app->linebuf->str, '\n'))) {
      *nl = '\0';
      handle_line(app, app->linebuf->str);
      g_string_erase(app->linebuf, 0, (nl - app->linebuf->str) + 1);
    }
  }
  if (e) g_clear_error(&e);

  if (cond & (G_IO_HUP | G_IO_ERR))
    return G_SOURCE_REMOVE;     /* pipe closed; child watch handles cleanup */
  return G_SOURCE_CONTINUE;
}

/* ── process lifecycle ────────────────────────────────────────────────────*/

static void clear_process(ZrApp *app) {
  if (app->out_watch)   { g_source_remove(app->out_watch); app->out_watch = 0; }
  if (app->out)         { g_io_channel_shutdown(app->out, FALSE, NULL);
                          g_io_channel_unref(app->out); app->out = NULL; }
  if (app->child_watch) { g_source_remove(app->child_watch); app->child_watch = 0; }
  if (app->pid)         { g_spawn_close_pid(app->pid); app->pid = 0; }
  g_string_truncate(app->linebuf, 0);
  app->persistent = FALSE;
  g_clear_pointer(&app->url, g_free);
  g_clear_pointer(&app->qr_path, g_free);
  /* drop the roster data only — rendering is done by callers that still have
   * live widgets (clear_process also runs from zr_app_free, after the panel has
   * already destroyed the popup's widget tree). */
  if (app->peers) g_hash_table_remove_all(app->peers);
}

static void on_child_exit(GPid pid, gint status, gpointer data) {
  ZrApp *app = data;
  (void) status;
  if (pid != app->pid) { g_spawn_close_pid(pid); return; }
  clear_process(app);
  render_clients(app);   /* widgets still live here — hide the roster */
  set_status(app, ZR_IDLE);
}

/* New session leader so we can signal the whole group on stop. */
static void child_setup(gpointer d) { (void) d; setsid(); }

void zr_app_stop(ZrApp *app) {
  if (!app->pid) return;
  /* signal the process group (negative pid); agent is its own session leader */
  kill(-app->pid, SIGTERM);
}

static void start_agent(ZrApp *app) {
  if (app->pid) return;

  if (!app->agent_path) app->agent_path = find_agent();
  if (!app->agent_path) {
    gtk_label_set_text(GTK_LABEL(app->status_lbl), zr_t("no_agent"));
    return;
  }

  /* remember only in remote mode, with a capable agent, and the box ticked */
  gboolean remember = app->agent_remember && app->remote_mode && app->remember_chk &&
      gtk_toggle_button_get_active(GTK_TOGGLE_BUTTON(app->remember_chk));

  char *argv[6];
  int n = 0;
  argv[n++] = app->agent_path;
  argv[n++] = (char*)"-machine";
  argv[n++] = (char*)"-mode";
  argv[n++] = (char*)(app->remote_mode ? "remote" : "lan");
  if (remember) argv[n++] = (char*)"-remember";
  argv[n] = NULL;

  GError *e = NULL;
  gint outfd = -1;
  gboolean ok = g_spawn_async_with_pipes(
      NULL, argv, NULL,
      G_SPAWN_DO_NOT_REAP_CHILD,
      child_setup, NULL,
      &app->pid,
      NULL, &outfd, NULL,   /* capture stdout; inherit stderr */
      &e);

  if (!ok) {
    gtk_label_set_text(GTK_LABEL(app->status_lbl),
                       e ? e->message : zr_t("no_agent"));
    if (e) g_clear_error(&e);
    app->pid = 0;
    return;
  }

  app->out = g_io_channel_unix_new(outfd);
  g_io_channel_set_flags(app->out, G_IO_FLAG_NONBLOCK, NULL);
  g_io_channel_set_encoding(app->out, NULL, NULL);  /* binary-safe */
  app->out_watch  = g_io_add_watch(app->out, G_IO_IN | G_IO_HUP | G_IO_ERR,
                                   on_agent_output, app);
  app->child_watch = g_child_watch_add(app->pid, on_child_exit, app);

  set_status(app, ZR_STARTING);
}

/* ── widget callbacks ─────────────────────────────────────────────────────*/

static void on_start_clicked(GtkButton *b, gpointer data) {
  (void) b;
  ZrApp *app = data;
  if (app->pid) zr_app_stop(app);
  else          start_agent(app);
}

static void on_mode_toggled(GtkToggleButton *b, gpointer data) {
  ZrApp *app = data;
  if (gtk_toggle_button_get_active(b)) {
    app->remote_mode = (GTK_WIDGET(b) == app->mode_remote);
    /* the "remember" option is meaningless on a LAN (the address is the
     * machine's local IP, not a relay room) — enable it only for remote, and
     * only when the resolved agent actually supports --remember. */
    if (app->remember_chk)
      gtk_widget_set_sensitive(app->remember_chk, app->remote_mode && app->agent_remember);
  }
}

static gboolean reset_copy_label(gpointer data) {
  ZrApp *app = data;
  gtk_label_set_text(GTK_LABEL(app->copy_lbl), zr_t("copy"));
  return G_SOURCE_REMOVE;
}

static void on_copy_clicked(GtkButton *b, gpointer data) {
  (void) b;
  ZrApp *app = data;
  if (!app->url) return;
  GtkClipboard *cb = gtk_clipboard_get(GDK_SELECTION_CLIPBOARD);
  gtk_clipboard_set_text(cb, app->url, -1);
  gtk_label_set_text(GTK_LABEL(app->copy_lbl), zr_t("copied"));
  g_timeout_add(1600, reset_copy_label, app);
}

/* ── widget construction ──────────────────────────────────────────────────*/

static GtkWidget *make_mode_radio(const char *title, const char *desc,
                                  GtkWidget *group_with) {
  GtkWidget *r = group_with
    ? gtk_radio_button_new_from_widget(GTK_RADIO_BUTTON(group_with))
    : gtk_radio_button_new(NULL);
  GtkWidget *box = gtk_box_new(GTK_ORIENTATION_VERTICAL, 0);
  GtkWidget *t = gtk_label_new(title);
  gtk_widget_set_halign(t, GTK_ALIGN_START);
  GtkWidget *d = gtk_label_new(desc);
  gtk_widget_set_halign(d, GTK_ALIGN_START);
  gtk_style_context_add_class(gtk_widget_get_style_context(d), "dim-label");
  PangoAttrList *al = pango_attr_list_new();
  pango_attr_list_insert(al, pango_attr_scale_new(0.85));
  gtk_label_set_attributes(GTK_LABEL(d), al);
  pango_attr_list_unref(al);
  gtk_box_pack_start(GTK_BOX(box), t, FALSE, FALSE, 0);
  gtk_box_pack_start(GTK_BOX(box), d, FALSE, FALSE, 0);
  gtk_container_add(GTK_CONTAINER(r), box);
  return r;
}

static void build_ui(ZrApp *app) {
  GtkWidget *root = gtk_box_new(GTK_ORIENTATION_VERTICAL, 12);
  gtk_container_set_border_width(GTK_CONTAINER(root), 16);
  gtk_widget_set_size_request(root, 280, -1);

  /* header */
  GtkWidget *title = gtk_label_new(NULL);
  char *mk = g_markup_printf_escaped(
      "<span weight='bold' size='large'>%s</span>", zr_t("title"));
  gtk_label_set_markup(GTK_LABEL(title), mk);
  g_free(mk);
  gtk_widget_set_halign(title, GTK_ALIGN_START);

  GtkWidget *tagline = gtk_label_new(zr_t("tagline"));
  gtk_widget_set_halign(tagline, GTK_ALIGN_START);
  gtk_style_context_add_class(gtk_widget_get_style_context(tagline), "dim-label");

  gtk_box_pack_start(GTK_BOX(root), title, FALSE, FALSE, 0);
  gtk_box_pack_start(GTK_BOX(root), tagline, FALSE, FALSE, 0);
  gtk_box_pack_start(GTK_BOX(root),
      gtk_separator_new(GTK_ORIENTATION_HORIZONTAL), FALSE, FALSE, 2);

  /* mode chooser */
  app->mode_lan = make_mode_radio(zr_t("mode_lan"), zr_t("mode_lan_d"), NULL);
  app->mode_remote = make_mode_radio(zr_t("mode_remote"), zr_t("mode_remote_d"),
                                     app->mode_lan);
  gtk_toggle_button_set_active(GTK_TOGGLE_BUTTON(app->mode_lan), TRUE);
  g_signal_connect(app->mode_lan, "toggled", G_CALLBACK(on_mode_toggled), app);
  g_signal_connect(app->mode_remote, "toggled", G_CALLBACK(on_mode_toggled), app);
  gtk_box_pack_start(GTK_BOX(root), app->mode_lan, FALSE, FALSE, 0);
  gtk_box_pack_start(GTK_BOX(root), app->mode_remote, FALSE, FALSE, 0);

  /* remember toggle — remote only (insensitive while LAN is selected) */
  app->remember_chk = gtk_check_button_new();
  {
    GtkWidget *cbx = gtk_box_new(GTK_ORIENTATION_VERTICAL, 0);
    GtkWidget *ct = gtk_label_new(zr_t("remember"));
    gtk_widget_set_halign(ct, GTK_ALIGN_START);
    GtkWidget *cd = gtk_label_new(zr_t("remember_d"));
    gtk_widget_set_halign(cd, GTK_ALIGN_START);
    gtk_style_context_add_class(gtk_widget_get_style_context(cd), "dim-label");
    PangoAttrList *cal = pango_attr_list_new();
    pango_attr_list_insert(cal, pango_attr_scale_new(0.85));
    gtk_label_set_attributes(GTK_LABEL(cd), cal);
    pango_attr_list_unref(cal);
    gtk_box_pack_start(GTK_BOX(cbx), ct, FALSE, FALSE, 0);
    gtk_box_pack_start(GTK_BOX(cbx), cd, FALSE, FALSE, 0);
    gtk_container_add(GTK_CONTAINER(app->remember_chk), cbx);
  }
  gtk_widget_set_sensitive(app->remember_chk, FALSE); /* LAN is the default */
  /* if the resolved agent is too old for --remember, say why on hover */
  if (!app->agent_remember)
    gtk_widget_set_tooltip_text(app->remember_chk, zr_t("remember_old"));
  gtk_box_pack_start(GTK_BOX(root), app->remember_chk, FALSE, FALSE, 0);

  /* start/stop */
  app->start_btn = gtk_button_new();
  app->start_lbl = gtk_label_new(zr_t("start"));
  gtk_container_add(GTK_CONTAINER(app->start_btn), app->start_lbl);
  gtk_style_context_add_class(gtk_widget_get_style_context(app->start_btn),
                              "suggested-action");
  g_signal_connect(app->start_btn, "clicked", G_CALLBACK(on_start_clicked), app);
  gtk_box_pack_start(GTK_BOX(root), app->start_btn, FALSE, FALSE, 0);

  /* status line */
  app->status_lbl = gtk_label_new(zr_t("st_idle"));
  gtk_label_set_line_wrap(GTK_LABEL(app->status_lbl), TRUE);
  gtk_label_set_justify(GTK_LABEL(app->status_lbl), GTK_JUSTIFY_CENTER);
  gtk_box_pack_start(GTK_BOX(root), app->status_lbl, FALSE, FALSE, 0);

  /* connected-clients roster (count + IP list) — hidden until someone connects */
  app->clients_lbl = gtk_label_new(NULL);
  gtk_label_set_justify(GTK_LABEL(app->clients_lbl), GTK_JUSTIFY_CENTER);
  gtk_widget_set_halign(app->clients_lbl, GTK_ALIGN_CENTER);
  gtk_widget_set_no_show_all(app->clients_lbl, TRUE);
  gtk_widget_set_visible(app->clients_lbl, FALSE);
  gtk_box_pack_start(GTK_BOX(root), app->clients_lbl, FALSE, FALSE, 0);

  /* pairing block (QR + url + copy) — hidden until waiting */
  app->pair_box = gtk_box_new(GTK_ORIENTATION_VERTICAL, 8);
  app->qr_img = gtk_image_new();
  gtk_widget_set_halign(app->qr_img, GTK_ALIGN_CENTER);
  /* white frame so dark panel themes don't break QR contrast */
  GtkWidget *qr_frame = gtk_frame_new(NULL);
  gtk_frame_set_shadow_type(GTK_FRAME(qr_frame), GTK_SHADOW_NONE);
  GtkWidget *qr_pad = gtk_event_box_new();
  GtkStyleContext *qc = gtk_widget_get_style_context(qr_pad);
  GtkCssProvider *css = gtk_css_provider_new();
  gtk_css_provider_load_from_data(css,
      "* { background:#ffffff; border-radius:8px; padding:8px; }", -1, NULL);
  gtk_style_context_add_provider(qc, GTK_STYLE_PROVIDER(css),
                                 GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
  g_object_unref(css);
  gtk_container_add(GTK_CONTAINER(qr_pad), app->qr_img);
  gtk_container_add(GTK_CONTAINER(qr_frame), qr_pad);
  gtk_widget_set_halign(qr_frame, GTK_ALIGN_CENTER);
  gtk_box_pack_start(GTK_BOX(app->pair_box), qr_frame, FALSE, FALSE, 0);

  /* "saved" hint — shown only when the agent reports a remembered identity */
  app->saved_lbl = gtk_label_new(zr_t("saved_hint"));
  gtk_label_set_line_wrap(GTK_LABEL(app->saved_lbl), TRUE);
  gtk_label_set_justify(GTK_LABEL(app->saved_lbl), GTK_JUSTIFY_CENTER);
  gtk_widget_set_halign(app->saved_lbl, GTK_ALIGN_CENTER);
  {
    PangoAttrList *sal = pango_attr_list_new();
    pango_attr_list_insert(sal, pango_attr_scale_new(0.85));
    gtk_label_set_attributes(GTK_LABEL(app->saved_lbl), sal);
    pango_attr_list_unref(sal);
  }
  gtk_style_context_add_class(gtk_widget_get_style_context(app->saved_lbl), "dim-label");
  gtk_widget_set_no_show_all(app->saved_lbl, TRUE);
  gtk_widget_set_visible(app->saved_lbl, FALSE);
  gtk_box_pack_start(GTK_BOX(app->pair_box), app->saved_lbl, FALSE, FALSE, 0);

  GtkWidget *hint = gtk_label_new(zr_t("open_phone"));
  gtk_style_context_add_class(gtk_widget_get_style_context(hint), "dim-label");
  PangoAttrList *al = pango_attr_list_new();
  pango_attr_list_insert(al, pango_attr_scale_new(0.85));
  gtk_label_set_attributes(GTK_LABEL(hint), al);
  pango_attr_list_unref(al);
  gtk_box_pack_start(GTK_BOX(app->pair_box), hint, FALSE, FALSE, 0);

  app->url_entry = gtk_entry_new();
  gtk_editable_set_editable(GTK_EDITABLE(app->url_entry), FALSE);
  gtk_widget_set_can_focus(app->url_entry, TRUE);
  gtk_box_pack_start(GTK_BOX(app->pair_box), app->url_entry, FALSE, FALSE, 0);

  app->copy_btn = gtk_button_new();
  app->copy_lbl = gtk_label_new(zr_t("copy"));
  gtk_container_add(GTK_CONTAINER(app->copy_btn), app->copy_lbl);
  g_signal_connect(app->copy_btn, "clicked", G_CALLBACK(on_copy_clicked), app);
  gtk_box_pack_start(GTK_BOX(app->pair_box), app->copy_btn, FALSE, FALSE, 0);

  gtk_box_pack_start(GTK_BOX(root), app->pair_box, FALSE, FALSE, 0);

  gtk_widget_show_all(root);
  /* realize the pair_box children once, then exempt the box from future
   * show_all (the panel re-runs it on popup) and hide it until pairing. */
  gtk_widget_set_no_show_all(app->pair_box, TRUE);
  gtk_widget_set_visible(app->pair_box, FALSE);
  app->root = root;
}

/* ── public API ───────────────────────────────────────────────────────────*/

ZrApp *zr_app_new(void) {
  ZrApp *app = g_new0(ZrApp, 1);
  app->linebuf = g_string_new(NULL);
  app->status = ZR_IDLE;
  app->peers = g_hash_table_new_full(g_direct_hash, g_direct_equal, NULL, g_free);
  app->agent_path = find_agent();
  app->agent_remember = probe_remember(app->agent_path);
  build_ui(app);
  return app;
}

GtkWidget *zr_app_widget(ZrApp *app) { return app->root; }
ZrStatus   zr_app_status(ZrApp *app) { return app->status; }

void zr_app_start(ZrApp *app, gboolean remote) {
  app->remote_mode = remote;
  gtk_toggle_button_set_active(
      GTK_TOGGLE_BUTTON(remote ? app->mode_remote : app->mode_lan), TRUE);
  start_agent(app);
}

void zr_app_set_status_cb(ZrApp *app, void (*cb)(ZrStatus, gpointer), gpointer user) {
  app->status_cb = cb;
  app->status_user = user;
}

void zr_app_free(ZrApp *app) {
  if (!app) return;
  zr_app_stop(app);
  clear_process(app);
  if (app->peers) g_hash_table_destroy(app->peers);
  g_string_free(app->linebuf, TRUE);
  g_free(app->agent_path);
  g_free(app);
}
