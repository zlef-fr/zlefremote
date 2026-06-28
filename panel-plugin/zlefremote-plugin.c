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
} PluginCtx;

static gboolean reposition_idle(gpointer data);

/* ── panel icon ───────────────────────────────────────────────────────────*/

static void set_icon(PluginCtx *ctx, gint size) {
  GtkIconTheme *theme = gtk_icon_theme_get_default();
  const char *name = gtk_icon_theme_has_icon(theme, "zlefremote")
                       ? "zlefremote" : "input-tablet";
  gtk_image_set_from_icon_name(GTK_IMAGE(ctx->icon), name, GTK_ICON_SIZE_BUTTON);
  gtk_image_set_pixel_size(GTK_IMAGE(ctx->icon),
                           size > 16 ? size - 6 : size);
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

/* (Re)place the popup against the panel button. The panel keeps it within the
 * monitor and flips it to the other side of the panel when it would overflow. */
static void popup_position(PluginCtx *ctx) {
  gint x = 0, y = 0;
  if (!gtk_widget_get_visible(ctx->popup)) return;
  xfce_panel_plugin_position_widget(ctx->plugin, ctx->popup, ctx->button, &x, &y);
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
  gtk_widget_show(ctx->popup);
  xfce_panel_plugin_block_autohide(ctx->plugin, TRUE);
  popup_position(ctx);
  gtk_window_present(GTK_WINDOW(ctx->popup));
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
