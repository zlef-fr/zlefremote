package fr.zlef.remote

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import android.support.v4.media.MediaMetadataCompat
import android.support.v4.media.session.MediaSessionCompat
import android.support.v4.media.session.PlaybackStateCompat
import androidx.core.app.NotificationCompat
import androidx.media.session.MediaButtonReceiver

/**
 * Keeps one remote-control session alive and puts the computer's transport
 * controls where you can reach them without unlocking: the notification shade
 * and the lock screen.
 *
 * The web client had to loop a silent audio file to obtain a media card, which
 * held audio focus and paused whatever the phone itself was playing — so it
 * shipped switched off by default. A real MediaSession needs no audio at all,
 * which is why this can be on out of the box.
 *
 * Buttons do not play anything here; they send the computer's media keys, which
 * is what the user is actually asking for.
 */
class ZrSessionService : Service() {

    companion object {
        private const val CHANNEL_ID = "zr_session"
        private const val NOTIFICATION_ID = 0x2E01

        private const val ACTION_START = "fr.zlef.remote.START"
        private const val ACTION_UPDATE = "fr.zlef.remote.UPDATE"
        private const val ACTION_STOP = "fr.zlef.remote.STOP"
        private const val EXTRA_HOST = "host"
        private const val EXTRA_CONNECTED = "connected"

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

    private var session: MediaSessionCompat? = null
    private var host: String = ""
    private var connected: Boolean = false
    private var started = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createChannel()
        session = MediaSessionCompat(this, "ZlefRemote").apply {
            setCallback(object : MediaSessionCompat.Callback() {
                override fun onPlay() = send("playpause")
                override fun onPause() = send("playpause")
                override fun onSkipToNext() = send("next")
                override fun onSkipToPrevious() = send("prev")
                override fun onStop() {
                    MainActivity.emit(mapOf("type" to "stop"))
                }
            })
            isActive = true
        }
        publishPlaybackState()
    }

    private fun send(key: String) {
        MainActivity.emit(mapOf("type" to "media", "key" to key))
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_STOP -> {
                stopSelf()
                return START_NOT_STICKY
            }

            ACTION_START, ACTION_UPDATE -> {
                host = intent.getStringExtra(EXTRA_HOST) ?: host
                connected = intent.getBooleanExtra(EXTRA_CONNECTED, connected)
            }
        }
        MediaButtonReceiver.handleIntent(session, intent)

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

    /**
     * The session is marked PLAYING so the card stays put — a paused session is
     * collected by the system after a moment, and the whole point is that these
     * buttons are there when you pick the phone up.
     */
    private fun publishPlaybackState() {
        session?.setPlaybackState(
            PlaybackStateCompat.Builder()
                .setActions(
                    PlaybackStateCompat.ACTION_PLAY_PAUSE or
                        PlaybackStateCompat.ACTION_PLAY or
                        PlaybackStateCompat.ACTION_PAUSE or
                        PlaybackStateCompat.ACTION_SKIP_TO_NEXT or
                        PlaybackStateCompat.ACTION_SKIP_TO_PREVIOUS or
                        PlaybackStateCompat.ACTION_STOP
                )
                .setState(PlaybackStateCompat.STATE_PLAYING, 0, 1f)
                .build()
        )
        session?.setMetadata(
            MediaMetadataCompat.Builder()
                .putString(
                    MediaMetadataCompat.METADATA_KEY_TITLE,
                    host.ifEmpty { getString(R.string.app_name) },
                )
                .putString(
                    MediaMetadataCompat.METADATA_KEY_ARTIST,
                    getString(R.string.session_subtitle),
                )
                .build()
        )
    }

    private fun buildNotification(): android.app.Notification {
        publishPlaybackState()

        val open = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java)
                .addFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        fun action(icon: Int, title: Int, playbackAction: Long) = NotificationCompat.Action(
            icon,
            getString(title),
            MediaButtonReceiver.buildMediaButtonPendingIntent(this, playbackAction),
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
            .setCategory(NotificationCompat.CATEGORY_TRANSPORT)
            .addAction(
                action(
                    R.drawable.ic_media_previous,
                    R.string.action_previous,
                    PlaybackStateCompat.ACTION_SKIP_TO_PREVIOUS,
                )
            )
            .addAction(
                action(
                    R.drawable.ic_media_playpause,
                    R.string.action_play_pause,
                    PlaybackStateCompat.ACTION_PLAY_PAUSE,
                )
            )
            .addAction(
                action(
                    R.drawable.ic_media_next,
                    R.string.action_next,
                    PlaybackStateCompat.ACTION_SKIP_TO_NEXT,
                )
            )
            .setStyle(
                androidx.media.app.NotificationCompat.MediaStyle()
                    .setMediaSession(session?.sessionToken)
                    .setShowActionsInCompactView(0, 1, 2)
            )
            .build()
    }

    override fun onDestroy() {
        session?.isActive = false
        session?.release()
        session = null
        started = false
        super.onDestroy()
    }
}
