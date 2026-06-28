/* Standalone host for the ZlefRemote popup — development / visual testing only.
 *
 * Builds the exact same UI the panel plugin shows, in a normal toplevel window,
 * so it can be driven and screenshotted without a running xfce4-panel.
 *
 *   ./zr-standalone            # interactive
 *   ZR_AUTOSHOT=/tmp/x.png ./zr-standalone   # render, grab, quit (under Xvfb)
 */
#include "zlefremote.h"

static gboolean autoshot(gpointer data) {
  GtkWidget *win = data;
  const char *path = g_getenv("ZR_AUTOSHOT");
  if (path) {
    GdkWindow *gw = gtk_widget_get_window(win);
    if (gw) {
      int w = gdk_window_get_width(gw), h = gdk_window_get_height(gw);
      GdkPixbuf *pb = gdk_pixbuf_get_from_window(gw, 0, 0, w, h);
      if (pb) { gdk_pixbuf_save(pb, path, "png", NULL, NULL); g_object_unref(pb); }
    }
    gtk_main_quit();
  }
  return G_SOURCE_REMOVE;
}

int main(int argc, char **argv) {
  gtk_init(&argc, &argv);

  ZrApp *app = zr_app_new();

  GtkWidget *win = gtk_window_new(GTK_WINDOW_TOPLEVEL);
  gtk_window_set_title(GTK_WINDOW(win), "ZlefRemote (standalone)");
  gtk_container_add(GTK_CONTAINER(win), zr_app_widget(app));
  g_signal_connect(win, "destroy", G_CALLBACK(gtk_main_quit), NULL);
  gtk_widget_show_all(win);

  /* ZR_AUTOSTART=lan|remote → kick off pairing so a screenshot shows the QR. */
  const char *as = g_getenv("ZR_AUTOSTART");
  if (as && *as)
    zr_app_start(app, g_ascii_strcasecmp(as, "remote") == 0);

  if (g_getenv("ZR_AUTOSHOT"))
    g_timeout_add(as && *as ? 3000 : 1200, autoshot, win);

  gtk_main();
  zr_app_free(app);
  return 0;
}
