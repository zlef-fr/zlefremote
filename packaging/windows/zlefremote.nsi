; ZlefRemote — Windows installer (NSIS 3, built on Linux with makensis).
;
; Deliberately a PER-USER install: the tray app is a per-user thing (its
; "start with Windows" switch and its settings live in HKCU), and installing
; into %LOCALAPPDATA%\Programs means no UAC prompt at all. Elevating would put
; the Run key and the Start Menu shortcut in the *administrator's* hive, which
; is the classic way tray apps end up never starting for the person who
; installed them.
;
; Build:  makensis -DVERSION=1.0.0 packaging/windows/zlefremote.nsi

Unicode true
!include "MUI2.nsh"
!include "FileFunc.nsh"

!ifndef VERSION
  !define VERSION "1.0.0"
!endif
!ifndef ARCH
  !define ARCH "amd64"
!endif

!define APPNAME     "ZlefRemote"
!define COMPANY     "zlef.fr"
!define WEBSITE     "https://remote.zlef.fr"
!define REGKEY      "Software\Microsoft\Windows\CurrentVersion\Uninstall\ZlefRemote"
!define RUNKEY      "Software\Microsoft\Windows\CurrentVersion\Run"
!define DIST        "..\..\dist"

Name "${APPNAME}"
OutFile "${DIST}\zlefremote-setup-windows-${ARCH}.exe"
RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Programs\ZlefRemote"
InstallDirRegKey HKCU "Software\ZlefRemote" "InstallDir"
SetCompressor /SOLID lzma
ShowInstDetails hide
ShowUnInstDetails hide

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName"     "${APPNAME}"
VIAddVersionKey "CompanyName"     "${COMPANY}"
VIAddVersionKey "FileDescription" "ZlefRemote — your phone is the trackpad"
VIAddVersionKey "FileVersion"     "${VERSION}"
VIAddVersionKey "ProductVersion"  "${VERSION}"
VIAddVersionKey "LegalCopyright"  "zlef.fr"

!define MUI_ICON   "..\..\tray\assets\zlefremote.ico"
!define MUI_UNICON "..\..\tray\assets\zlefremote.ico"
!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\ZlefRemote.exe"
!define MUI_FINISHPAGE_RUN_TEXT "$(RUN_TEXT)"
!define MUI_FINISHPAGE_LINK "remote.zlef.fr"
!define MUI_FINISHPAGE_LINK_LOCATION "${WEBSITE}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Two locales, resolved from the OS — same rule as the app itself.
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "French"

LangString RUN_TEXT      ${LANG_ENGLISH} "Start ZlefRemote now"
LangString RUN_TEXT      ${LANG_FRENCH}  "Lancer ZlefRemote maintenant"
LangString SEC_CORE      ${LANG_ENGLISH} "ZlefRemote (required)"
LangString SEC_CORE      ${LANG_FRENCH}  "ZlefRemote (obligatoire)"
LangString SEC_CORE_D    ${LANG_ENGLISH} "The tray app and the agent that talks to your phone."
LangString SEC_CORE_D    ${LANG_FRENCH}  "L'application de la zone de notification et l'agent qui parle à votre téléphone."
LangString SEC_START     ${LANG_ENGLISH} "Start menu shortcut"
LangString SEC_START     ${LANG_FRENCH}  "Raccourci dans le menu Démarrer"
LangString SEC_DESKTOP   ${LANG_ENGLISH} "Desktop shortcut"
LangString SEC_DESKTOP   ${LANG_FRENCH}  "Raccourci sur le Bureau"
LangString SEC_AUTO      ${LANG_ENGLISH} "Start with Windows"
LangString SEC_AUTO      ${LANG_FRENCH}  "Démarrer avec Windows"
LangString SEC_AUTO_D    ${LANG_ENGLISH} "Put the ZlefRemote icon in the notification area at every sign-in."
LangString SEC_AUTO_D    ${LANG_FRENCH}  "Placer l'icône ZlefRemote dans la zone de notification à chaque ouverture de session."
LangString RUNNING       ${LANG_ENGLISH} "Closing the running ZlefRemote…"
LangString RUNNING       ${LANG_FRENCH}  "Fermeture de ZlefRemote en cours d'exécution…"

Function .onInit
  !insertmacro MUI_LANGDLL_DISPLAY
FunctionEnd

; Stop a running instance (and its agent) before overwriting the binaries.
!macro KillRunning
  DetailPrint "$(RUNNING)"
  nsExec::Exec 'taskkill /F /IM ZlefRemote.exe'
  nsExec::Exec 'taskkill /F /IM zlefremote-agent.exe'
  Sleep 400
!macroend

Section "$(SEC_CORE)" SecCore
  SectionIn RO
  !insertmacro KillRunning

  SetOutPath "$INSTDIR"
  File /oname=ZlefRemote.exe        "${DIST}\zlefremote-tray-windows-${ARCH}.exe"
  File /oname=zlefremote-agent.exe  "${DIST}\zlefremote-agent-windows-${ARCH}.exe"
  File /oname=README.txt            "README-windows.txt"

  WriteRegStr HKCU "Software\ZlefRemote" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Programs & Features entry (per-user hive, matching the install)
  WriteRegStr   HKCU "${REGKEY}" "DisplayName"     "${APPNAME}"
  WriteRegStr   HKCU "${REGKEY}" "DisplayVersion"  "${VERSION}"
  WriteRegStr   HKCU "${REGKEY}" "Publisher"       "${COMPANY}"
  WriteRegStr   HKCU "${REGKEY}" "DisplayIcon"     "$INSTDIR\ZlefRemote.exe"
  WriteRegStr   HKCU "${REGKEY}" "URLInfoAbout"    "${WEBSITE}"
  WriteRegStr   HKCU "${REGKEY}" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegStr   HKCU "${REGKEY}" "InstallLocation" "$INSTDIR"
  WriteRegDWORD HKCU "${REGKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${REGKEY}" "NoRepair" 1
  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKCU "${REGKEY}" "EstimatedSize" "$0"
SectionEnd

Section "$(SEC_START)" SecStart
  CreateDirectory "$SMPROGRAMS\ZlefRemote"
  CreateShortcut  "$SMPROGRAMS\ZlefRemote\ZlefRemote.lnk" "$INSTDIR\ZlefRemote.exe"
  CreateShortcut  "$SMPROGRAMS\ZlefRemote\Uninstall ZlefRemote.lnk" "$INSTDIR\uninstall.exe"
SectionEnd

Section /o "$(SEC_DESKTOP)" SecDesktop
  CreateShortcut "$DESKTOP\ZlefRemote.lnk" "$INSTDIR\ZlefRemote.exe"
SectionEnd

Section "$(SEC_AUTO)" SecAuto
  WriteRegStr HKCU "${RUNKEY}" "ZlefRemote" '"$INSTDIR\ZlefRemote.exe"'
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecCore} "$(SEC_CORE_D)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecAuto} "$(SEC_AUTO_D)"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "Uninstall"
  !insertmacro KillRunning

  Delete "$INSTDIR\ZlefRemote.exe"
  Delete "$INSTDIR\zlefremote-agent.exe"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\uninstall.exe"
  RMDir  "$INSTDIR"

  Delete "$SMPROGRAMS\ZlefRemote\ZlefRemote.lnk"
  Delete "$SMPROGRAMS\ZlefRemote\Uninstall ZlefRemote.lnk"
  RMDir  "$SMPROGRAMS\ZlefRemote"
  Delete "$DESKTOP\ZlefRemote.lnk"

  DeleteRegValue HKCU "${RUNKEY}" "ZlefRemote"
  DeleteRegKey   HKCU "${REGKEY}"
  ; App preferences go too — but never the agent's saved identity in
  ; %APPDATA%\zlefremote, so a reinstall keeps saved phones reconnecting.
  DeleteRegKey   HKCU "Software\ZlefRemote"
SectionEnd
