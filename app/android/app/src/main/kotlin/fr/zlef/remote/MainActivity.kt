package fr.zlef.remote

import android.Manifest
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.provider.Settings
import android.view.KeyEvent
import android.view.WindowManager
import androidx.core.content.ContextCompat
import androidx.core.content.FileProvider
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.EventChannel
import io.flutter.plugin.common.MethodChannel
import java.io.File

/**
 * The native half of ZlefRemote.
 *
 * Everything here exists because a browser could not do it: keeping the link
 * alive with the screen off, putting the computer's transport controls in the
 * shade and on the lock screen without stealing audio focus, reading the volume
 * rocker, and installing our own updates.
 */
class MainActivity : FlutterActivity() {

    companion object {
        private const val METHOD_CHANNEL = "fr.zlef.remote/native"
        private const val EVENT_CHANNEL = "fr.zlef.remote/events"
        private const val NOTIFICATION_REQUEST = 4711

        /** Shared with [ZrSessionService] so media buttons reach Dart. */
        @Volatile
        var events: EventChannel.EventSink? = null

        fun emit(payload: Map<String, Any?>) {
            events?.success(payload)
        }
    }

    private var captureVolumeKeys = false
    private var notificationResult: MethodChannel.Result? = null

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        EventChannel(flutterEngine.dartExecutor.binaryMessenger, EVENT_CHANNEL)
            .setStreamHandler(object : EventChannel.StreamHandler {
                override fun onListen(arguments: Any?, sink: EventChannel.EventSink?) {
                    events = sink
                }

                override fun onCancel(arguments: Any?) {
                    events = null
                }
            })

        MethodChannel(flutterEngine.dartExecutor.binaryMessenger, METHOD_CHANNEL)
            .setMethodCallHandler { call, result ->
                when (call.method) {
                    "startSession" -> {
                        ZrSessionService.start(this, call.argument<String>("host") ?: "")
                        result.success(true)
                    }

                    "updateSession" -> {
                        ZrSessionService.update(
                            this,
                            call.argument<String>("host") ?: "",
                            call.argument<Boolean>("connected") ?: false,
                        )
                        result.success(true)
                    }

                    "stopSession" -> {
                        ZrSessionService.stop(this)
                        result.success(true)
                    }

                    "setKeepAwake" -> {
                        val on = call.argument<Boolean>("on") ?: false
                        runOnUiThread {
                            if (on) {
                                window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
                            } else {
                                window.clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
                            }
                        }
                        result.success(true)
                    }

                    "setVolumeKeyCapture" -> {
                        captureVolumeKeys = call.argument<Boolean>("on") ?: false
                        result.success(true)
                    }

                    "hasNotificationPermission" -> result.success(hasNotificationPermission())

                    "requestNotificationPermission" -> {
                        if (hasNotificationPermission()) {
                            result.success(true)
                        } else {
                            notificationResult = result
                            requestPermissions(
                                arrayOf(Manifest.permission.POST_NOTIFICATIONS),
                                NOTIFICATION_REQUEST,
                            )
                        }
                    }

                    "canInstallPackages" -> result.success(canInstallPackages())

                    "installApk" -> {
                        val path = call.argument<String>("path")
                        result.success(path != null && installApk(path))
                    }

                    else -> result.notImplemented()
                }
            }
    }

    private fun hasNotificationPermission(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
        return ContextCompat.checkSelfPermission(
            this,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray,
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        if (requestCode == NOTIFICATION_REQUEST) {
            notificationResult?.success(hasNotificationPermission())
            notificationResult = null
        }
    }

    private fun canInstallPackages(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return true
        if (packageManager.canRequestPackageInstalls()) return true
        // send the user straight to the one toggle that unblocks it
        startActivity(
            Intent(
                Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                Uri.parse("package:$packageName"),
            ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        )
        return false
    }

    private fun installApk(path: String): Boolean {
        val file = File(path)
        if (!file.exists()) return false
        return try {
            val uri = FileProvider.getUriForFile(this, "$packageName.fileprovider", file)
            startActivity(
                Intent(Intent.ACTION_VIEW)
                    .setDataAndType(uri, "application/vnd.android.package-archive")
                    .addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            )
            true
        } catch (_: Exception) {
            false
        }
    }

    /**
     * Volume rocker → the computer's volume. Intercepted before the system sees
     * it, and only while the user asked for it; otherwise the phone behaves
     * exactly as it always does.
     */
    override fun dispatchKeyEvent(event: KeyEvent): Boolean {
        val isVolumeKey = event.keyCode == KeyEvent.KEYCODE_VOLUME_UP ||
            event.keyCode == KeyEvent.KEYCODE_VOLUME_DOWN
        if (captureVolumeKeys && isVolumeKey) {
            if (event.action == KeyEvent.ACTION_DOWN) {
                emit(
                    mapOf(
                        "type" to if (event.keyCode == KeyEvent.KEYCODE_VOLUME_UP) {
                            "volumeUp"
                        } else {
                            "volumeDown"
                        }
                    )
                )
            }
            // swallow the key-up as well, or the system shows its own slider
            return true
        }
        return super.dispatchKeyEvent(event)
    }
}
