/* ZlefRemote — xfce4-panel plugin glue.
 *
 * Registers an internal (GModule) panel plugin: a toggle button in the panel
 * that pops up the ZlefRemote control surface (zlefremote-ui.c). All the agent
 * lifecycle / pairing logic lives in the UI module; this file is only the
 * panel button, the popup window, and positioning/auto-hide plumbing.
 */
#include "zlefremote.h"

#include <libxfce4panel/libxfce4panel.h>

typedef struct {
  XfcePanelPlugin *plugin;
  GtkWidget *button;   /* toggle button shown in the panel */
  GtkWidget *icon;     /* its image */
  GtkWidget *popup;    /* the control window */
  ZrApp     *app;
  guint      reposition_src;  /* pending deferred reposition, 0 if none */
  gint       want_x, want_y;  /* where the popup belongs while it is shown */
  gint       corrections;     /* window-manager overrides undone this showing */
} PluginCtx;

static gboolean reposition_idle(gpointer data);

/* ── panel icon ───────────────────────────────────────────────────────────*/

static void set_icon(PluginCtx *ctx, gint size) {
  gint px = size > 16 ? size - 6 : size;

  /* Load our bundled PNG straight from disk. PNG decoding is built into
   * gdk-pixbuf, so the panel button always renders — independent of the icon
   * cache, the active icon theme, and (crucially) the host's SVG loader.
   *
   * We deliberately do NOT go through the icon theme here: for a non-exact size
   * GTK prefers a *scalable* (SVG) icon over our fixed-size PNGs, which would
   * re-invoke the SVG loader. On hosts where that loader is missing or broken
   * (e.g. a flaky librsvg/glycin) that leaves a broken icon — or crashes the
   * plugin wrapper when GTK then can't load its own SVG image-missing icon. */
  static const char *png[] = {
    "/usr/share/icons/hicolor/64x64/apps/zlefremote.png",
    "/usr/share/icons/hicolor/48x48/apps/zlefremote.png",
    "/usr/local/share/icons/hicolor/64x64/apps/zlefremote.png",
    "/usr/local/share/icons/hicolor/48x48/apps/zlefremote.png", NULL };
  for (int i = 0; png[i]; i++) {
    GdkPixbuf *pb = gdk_pixbuf_new_from_file_at_size(png[i], px, px, NULL);
    if (pb) {
      gtk_image_set_from_pixbuf(GTK_IMAGE(ctx->icon), pb);
      g_object_unref(pb);
      gtk_image_set_pixel_size(GTK_IMAGE(ctx->icon), px);
      return;
    }
  }

  /* PNGs not found (unusual install) — fall back to the themed icon by name */
  GtkIconTheme *theme = gtk_icon_theme_get_default();
  const char *name = gtk_icon_theme_has_icon(theme, "zlefremote")
                       ? "zlefremote" : "input-tablet";
  gtk_image_set_from_icon_name(GTK_IMAGE(ctx->icon), name, GTK_ICON_SIZE_BUTTON);
  gtk_image_set_pixel_size(GTK_IMAGE(ctx->icon), px);
}

static void on_status(ZrStatus st, gpointer data) {
  PluginCtx *ctx = data;
  GtkStyleContext *sc = gtk_widget_get_style_context(ctx->button);
  const char *tip = "ZlefRemote";
  switch (st) {
    case ZR_IDLE:     tip = "ZlefRemote"; break;
    case ZR_STARTING: tip = "ZlefRemote — starting…"; break;
    case ZR_WAITING:  tip = "ZlefRemote — waiting for a phone"; break;
    case ZR_PAIRED:   tip = "ZlefRemote — phone connected"; break;
  }
  gtk_widget_set_tooltip_text(ctx->button, tip);
  if (st == ZR_IDLE) gtk_style_context_remove_class(sc, "zr-active");
  else               gtk_style_context_add_class(sc, "zr-active");

  /* the popup grows when the QR/pairing block appears (and shrinks on stop) —
   * reposition once the new size is applied so it never spills off-screen */
  if (gtk_widget_get_visible(ctx->popup) && ctx->reposition_src == 0)
    ctx->reposition_src = g_timeout_add(10, reposition_idle, ctx);
}

/* ── popup show / hide ────────────────────────────────────────────────────*/

static void popup_hide(PluginCtx *ctx) {
  gtk_widget_hide(ctx->popup);
  xfce_panel_plugin_block_autohide(ctx->plugin, FALSE);
  if (gtk_toggle_button_get_active(GTK_TOGGLE_BUTTON(ctx->button)))
    gtk_toggle_button_set_active(GTK_TOGGLE_BUTTON(ctx->button), FALSE);
}

/* Where the popup belongs: against the panel button, on the side away from the
 * screen edge, kept inside the button's monitor.
 *
 * Computed from the button's LIVE root coordinates (an X round trip) and the
 * monitor under it, never from cached window state. xfce_panel_plugin_
 * position_widget() trusts gtk_window_get_position() on the plugin's toplevel
 * — in the external wrapper a GtkPlug, whose frame-extents lookup falls back to
 * stale cached coordinates — and after hours of suspend/resume and display
 * changes it put the popup far from the panel, out of reach until the panel
 * was restarted. It stays only as the fallback when the button is unmapped. */
static gboolean popup_compute(PluginCtx *ctx, gint *x, gint *y) {
  GtkWidget *top = gtk_widget_get_toplevel(ctx->button);
  GdkWindow *topwin = top ? gtk_widget_get_window(top) : NULL;
  gint tx = 0, ty = 0, rx = 0, ry = 0;
  if (!topwin || !gtk_widget_get_mapped(ctx->button) ||
      !gtk_widget_translate_coordinates(ctx->button, top, 0, 0, &tx, &ty)) {
    xfce_panel_plugin_position_widget(ctx->plugin, ctx->popup, ctx->button, x, y);
    return TRUE;
  }
  gdk_window_get_origin(topwin, &rx, &ry);

  GtkAllocation b;
  gtk_widget_get_allocation(ctx->button, &b);
  gint bx = rx + tx, by = ry + ty;

  GtkRequisition req;
  gtk_widget_get_preferred_size(ctx->popup, NULL, &req);

  GdkMonitor *mon = gdk_display_get_monitor_at_point(
      gtk_widget_get_display(ctx->button), bx + b.width / 2, by + b.height / 2);
  if (!mon) return FALSE;
  GdkRectangle g;
  gdk_monitor_get_geometry(mon, &g);

  switch (xfce_panel_plugin_arrow_type(ctx->plugin)) {
    case GTK_ARROW_UP:    *x = bx;             *y = by - req.height; break;
    case GTK_ARROW_DOWN:  *x = bx;             *y = by + b.height;   break;
    case GTK_ARROW_LEFT:  *x = bx - req.width; *y = by;              break;
    default:              *x = bx + b.width;   *y = by;              break;
  }
  *x = CLAMP(*x, g.x, MAX(g.x, g.x + g.width  - req.width));
  *y = CLAMP(*y, g.y, MAX(g.y, g.y + g.height - req.height));
  return TRUE;
}

/* (Re)place the popup against the panel button. Called before the popup maps
 * (so the window manager is handed a position and never applies its own dialog
 * placement) and again whenever its size or the window manager moves it. */
static void popup_position(PluginCtx *ctx) {
  gint x = 0, y = 0;
  if (!popup_compute(ctx, &x, &y)) return;
  ctx->want_x = x;
  ctx->want_y = y;
  gtk_window_move(GTK_WINDOW(ctx->popup), x, y);
}

/* Run after the popup's content has grown/shrunk (e.g. the QR appears on Start):
 * GTK applies the new natural size on the next main-loop pass, so we reposition
 * one idle tick later — otherwise the larger window keeps the old top-left and
 * spills off-screen until it is reopened. */
static gboolean reposition_idle(gpointer data) {
  PluginCtx *ctx = data;
  ctx->reposition_src = 0;
  popup_position(ctx);
  return G_SOURCE_REMOVE;
}

static void popup_show(PluginCtx *ctx) {
  ctx->corrections = 0;
  popup_position(ctx);            /* unmapped: becomes the map-time position */
  gtk_widget_show(ctx->popup);
  xfce_panel_plugin_block_autohide(ctx->plugin, TRUE);
  popup_position(ctx);
  gtk_window_present(GTK_WINDOW(ctx->popup));
}

/* The window manager has the last word on where a mapped window goes, and a
 * dialog it decides to place itself lands wherever its placement policy says —
 * typically the middle of the screen. Whenever the popup is seen anywhere but
 * where it belongs, put it back (bounded, so we never fight a WM that refuses). */
static gboolean on_popup_configure(GtkWidget *w, GdkEventConfigure *e, PluginCtx *ctx) {
  (void) w;
  if (!gtk_widget_get_visible(ctx->popup) || ctx->reposition_src) return FALSE;
  if (ABS(e->x - ctx->want_x) <= 2 && ABS(e->y - ctx->want_y) <= 2) return FALSE;
  if (ctx->corrections >= 5) return FALSE;
  ctx->corrections++;
  ctx->reposition_src = g_timeout_add(10, reposition_idle, ctx);
  return FALSE;
}

static gboolean on_popup_map(GtkWidget *w, GdkEvent *e, PluginCtx *ctx) {
  (void) w; (void) e;
  if (ctx->reposition_src == 0)
    ctx->reposition_src = g_timeout_add(10, reposition_idle, ctx);
  return FALSE;
}

static void on_button_toggled(GtkWidget *b, PluginCtx *ctx) {
  if (gtk_toggle_button_get_active(GTK_TOGGLE_BUTTON(b)))
    popup_show(ctx);
  else
    popup_hide(ctx);
}

static gboolean on_popup_focus_out(GtkWidget *w, GdkEvent *e, PluginCtx *ctx) {
  (void) w; (void) e;
  popup_hide(ctx);
  return FALSE;
}

static gboolean on_popup_key(GtkWidget *w, GdkEventKey *e, PluginCtx *ctx) {
  (void) w;
  if (e->keyval == GDK_KEY_Escape) { popup_hide(ctx); return TRUE; }
  return FALSE;
}

/* ── panel plugin signals ─────────────────────────────────────────────────*/

static gboolean on_size_changed(XfcePanelPlugin *plugin, gint size, PluginCtx *ctx) {
  (void) plugin;
  set_icon(ctx, size);
  return TRUE;
}

static void on_free(XfcePanelPlugin *plugin, PluginCtx *ctx) {
  (void) plugin;
  if (ctx->reposition_src) g_source_remove(ctx->reposition_src);
  if (ctx->popup) gtk_widget_destroy(ctx->popup);
  zr_app_free(ctx->app);
  g_free(ctx);
}

static void zlefremote_construct(XfcePanelPlugin *plugin) {
  PluginCtx *ctx = g_new0(PluginCtx, 1);
  ctx->plugin = plugin;
  ctx->app = zr_app_new();

  /* panel button */
  ctx->button = xfce_panel_create_toggle_button();
  ctx->icon = gtk_image_new();
  gtk_container_add(GTK_CONTAINER(ctx->button), ctx->icon);
  gtk_widget_set_tooltip_text(ctx->button, "ZlefRemote");
  set_icon(ctx, xfce_panel_plugin_get_size(plugin));
  gtk_container_add(GTK_CONTAINER(plugin), ctx->button);
  xfce_panel_plugin_add_action_widget(plugin, ctx->button);
  gtk_widget_show_all(ctx->button);
  g_signal_connect(ctx->button, "toggled", G_CALLBACK(on_button_toggled), ctx);

  /* popup window hosting the control surface */
  ctx->popup = gtk_window_new(GTK_WINDOW_TOPLEVEL);
  gtk_window_set_decorated(GTK_WINDOW(ctx->popup), FALSE);
  gtk_window_set_skip_taskbar_hint(GTK_WINDOW(ctx->popup), TRUE);
  gtk_window_set_skip_pager_hint(GTK_WINDOW(ctx->popup), TRUE);
  gtk_window_set_type_hint(GTK_WINDOW(ctx->popup), GDK_WINDOW_TYPE_HINT_DIALOG);
  gtk_window_set_resizable(GTK_WINDOW(ctx->popup), FALSE);
  gtk_window_set_transient_for(GTK_WINDOW(ctx->popup), NULL);
  gtk_container_add(GTK_CONTAINER(ctx->popup), zr_app_widget(ctx->app));
  g_signal_connect(ctx->popup, "focus-out-event",
                   G_CALLBACK(on_popup_focus_out), ctx);
  g_signal_connect(ctx->popup, "key-press-event",
                   G_CALLBACK(on_popup_key), ctx);
  g_signal_connect(ctx->popup, "configure-event",
                   G_CALLBACK(on_popup_configure), ctx);
  g_signal_connect(ctx->popup, "map-event",
                   G_CALLBACK(on_popup_map), ctx);
  /* never let destroy of the popup take down the agent before free-data */
  g_signal_connect(ctx->popup, "delete-event",
                   G_CALLBACK(gtk_widget_hide_on_delete), NULL);

  zr_app_set_status_cb(ctx->app, on_status, ctx);

  /* panel signals */
  g_signal_connect(plugin, "free-data", G_CALLBACK(on_free), ctx);
  g_signal_connect(plugin, "size-changed", G_CALLBACK(on_size_changed), ctx);
  xfce_panel_plugin_menu_show_about(plugin);
}

XFCE_PANEL_PLUGIN_REGISTER(zlefremote_construct)
