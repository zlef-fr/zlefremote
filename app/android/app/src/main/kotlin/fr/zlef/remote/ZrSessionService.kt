package fr.zlef.remote

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat

/**
 * Keeps one remote-control session alive and gives it a presence you can reach
 * without unlocking: an ongoing notification, plus [MainActivity] declared
 * `showWhenLocked`, so the remote appears over the keyguard the way a
 * navigation app does.
 *
 * Deliberately **not** a MediaSession. The web client faked one (silent looping
 * audio) to borrow the lock-screen music card, which stole audio focus and
 * paused whatever the phone was playing; the first native cut used a real
 * MediaSession, which is honest but still puts the computer's controls in the
 * *music* slot, competing with actual players and disappearing when one starts.
 * The controls here are plain notification actions routed straight to Dart, and
 * the lock-screen surface is our own activity.
 */
class ZrSessionService : Service() {

    companion object {
        private const val CHANNEL_ID = "zr_session"
        private const val NOTIFICATION_ID = 0x2E01

        private const val ACTION_START = "fr.zlef.remote.START"
        private const val ACTION_UPDATE = "fr.zlef.remote.UPDATE"
        private const val ACTION_STOP = "fr.zlef.remote.STOP"
        private const val ACTION_MEDIA = "fr.zlef.remote.MEDIA"
        private const val EXTRA_HOST = "host"
        private const val EXTRA_CONNECTED = "connected"
        private const val EXTRA_MEDIA_KEY = "key"

        fun start(context: Context, host: String) {
            val intent = Intent(context, ZrSessionService::class.java)
                .setAction(ACTION_START)
                .putExtra(EXTRA_HOST, host)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun update(context: Context, host: String, connected: Boolean) {
            context.startService(
                Intent(context, ZrSessionService::class.java)
                    .setAction(ACTION_UPDATE)
                    .putExtra(EXTRA_HOST, host)
                    .putExtra(EXTRA_CONNECTED, connected)
            )
        }

        fun stop(context: Context) {
            context.startService(
                Intent(context, ZrSessionService::class.java).setAction(ACTION_STOP)
            )
        }
    }

    private var host: String = ""
    private var connected: Boolean = false
    private var started = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createChannel()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopSelf()
                return START_NOT_STICKY
            }

            ACTION_MEDIA -> {
                intent.getStringExtra(EXTRA_MEDIA_KEY)?.let {
                    MainActivity.emit(mapOf("type" to "media", "key" to it))
                }
                return START_NOT_STICKY
            }

            ACTION_START, ACTION_UPDATE -> {
                host = intent.getStringExtra(EXTRA_HOST) ?: host
                connected = intent.getBooleanExtra(EXTRA_CONNECTED, connected)
            }
        }

        val notification = buildNotification()
        if (!started) {
            startForeground(NOTIFICATION_ID, notification)
            started = true
        } else {
            (getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
                .notify(NOTIFICATION_ID, notification)
        }
        return START_NOT_STICKY
    }

    private fun createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val channel = NotificationChannel(
            CHANNEL_ID,
            getString(R.string.session_channel_name),
            NotificationManager.IMPORTANCE_LOW, // present, never noisy
        ).apply {
            description = getString(R.string.session_channel_description)
            setShowBadge(false)
            lockscreenVisibility = NotificationCompat.VISIBILITY_PUBLIC
        }
        (getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager)
            .createNotificationChannel(channel)
    }

    /** A notification action that forwards one media verb to the session. */
    private fun mediaAction(icon: Int, title: Int, key: String): NotificationCompat.Action {
        val intent = Intent(this, ZrSessionService::class.java)
            .setAction(ACTION_MEDIA)
            .putExtra(EXTRA_MEDIA_KEY, key)
        val pending = PendingIntent.getService(
            this,
            key.hashCode(),
            intent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )
        return NotificationCompat.Action(icon, getString(title), pending)
    }

    private fun buildNotification(): android.app.Notification {
        // Opening from the shade or the lock screen lands on the remote itself;
        // MainActivity is showWhenLocked, so this works without unlocking.
        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java)
                .setAction(Intent.ACTION_MAIN)
                .addCategory(Intent.CATEGORY_LAUNCHER)
                .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_stat_zlefremote)
            .setContentTitle(host.ifEmpty { getString(R.string.app_name) })
            .setContentText(
                getString(
                    if (connected) R.string.session_connected else R.string.session_connecting
                )
            )
            .setContentIntent(open)
            .setOngoing(true)
            .setSilent(true)
            .setShowWhen(false)
            .setVisibility(NotificationCompat.VISIBILITY_PUBLIC)
            .setCategory(NotificationCompat.CATEGORY_SERVICE)
            .addAction(
                mediaAction(
                    R.drawable.ic_media_previous,
                    R.string.action_previous,
                    "prev",
                )
            )
            .addAction(
                mediaAction(
                    R.drawable.ic_media_playpause,
                    R.string.action_play_pause,
                    "playpause",
                )
            )
            .addAction(
                mediaAction(R.drawable.ic_media_next, R.string.action_next, "next")
            )
            .build()
    }

    override fun onDestroy() {
        started = false
        super.onDestroy()
    }
}
