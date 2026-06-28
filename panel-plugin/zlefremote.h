/* ZlefRemote xfce4-panel plugin — shared interface.
 *
 * The UI + agent-control logic lives in zlefremote-ui.c and depends only on
 * GTK3, so it can be hosted either inside the panel (zlefremote-plugin.c) or in
 * a standalone window for development/testing (zlefremote-standalone.c).
 */
#ifndef ZLEFREMOTE_H
#define ZLEFREMOTE_H

#include <gtk/gtk.h>

G_BEGIN_DECLS

typedef enum {
  ZR_IDLE,      /* agent not running                       */
  ZR_STARTING,  /* agent spawned, no pairing URL yet       */
  ZR_WAITING,   /* URL/QR ready, waiting for a phone       */
  ZR_PAIRED     /* a phone completed the handshake         */
} ZrStatus;

typedef struct _ZrApp ZrApp;

/* Lifecycle */
ZrApp     *zr_app_new       (void);
void       zr_app_free      (ZrApp *app);

/* The popup content (a GtkBox). Pack it into a panel popup window or a
 * standalone toplevel. Owned by the ZrApp. */
GtkWidget *zr_app_widget    (ZrApp *app);

/* Current state + a callback fired on every status change (used by the panel
 * button to reflect status as an icon/tooltip). */
ZrStatus   zr_app_status    (ZrApp *app);
void       zr_app_set_status_cb (ZrApp *app,
                                 void (*cb)(ZrStatus st, gpointer user),
                                 gpointer user);

/* Start the agent (remote=TRUE for relay mode, FALSE for LAN). Mainly used by
 * the standalone test harness; the popup's Start button is the normal path. */
void       zr_app_start     (ZrApp *app, gboolean remote);

/* Stop the agent if running (call before freeing / on plugin removal). */
void       zr_app_stop      (ZrApp *app);

/* Tiny built-in i18n (EN default + FR, auto-detected from the locale). */
const char *zr_t (const char *key);

G_END_DECLS

#endif /* ZLEFREMOTE_H */
