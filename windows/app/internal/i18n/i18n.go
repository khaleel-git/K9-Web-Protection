// Package i18n is a tiny message catalog; lookup order: active language → "en" → the key.
package i18n

import (
	"fmt"
	"sync"
)

// DefaultLang is used when no language has been configured.
const DefaultLang = "en"

// FallbackLang always exists in the catalog and is used for missing keys.
const FallbackLang = "en"

var (
	mu      sync.RWMutex
	current = DefaultLang
)

// SetLang selects the active language. An unknown or empty language falls back
// to FallbackLang.
func SetLang(lang string) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := catalog[lang]; ok {
		current = lang
		return
	}
	current = FallbackLang
}

// Lang returns the active language code (suitable for an HTML lang attribute).
func Lang() string {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// Dir returns the text direction of the active language ("rtl" or "ltr"),
// suitable for an HTML dir attribute.
func Dir() string {
	if Lang() == "he" {
		return "rtl"
	}
	return "ltr"
}

// T returns the message for key in the active language. Any args are applied
// with fmt.Sprintf. Missing keys fall back to English and then to the key.
func T(key string, args ...any) string {
	mu.RLock()
	lang := current
	mu.RUnlock()
	return TIn(lang, key, args...)
}

// TIn is T for an explicitly named language, regardless of the active one.
func TIn(lang, key string, args ...any) string {
	s, ok := catalog[lang][key]
	if !ok || s == "" {
		s, ok = catalog[FallbackLang][key]
	}
	if !ok || s == "" {
		return key // no catalog entry: never Sprintf over the raw key
	}
	if len(args) > 0 {
		return fmt.Sprintf(s, args...)
	}
	return s
}

// catalog maps language code → message key → text; keep every fmt verb in place and in order.
var catalog = map[string]map[string]string{
	"en": {
		// ── Application chrome ────────────────────────────────────────────
		"app.title":       "K10 Web Protection",
		"app.productName": "K10 Web Protection",

		// ── app.go errors surfaced as UI toasts ───────────────────────────
		"err.incorrectPassword":        "incorrect password",
		"err.incorrectCurrentPassword": "incorrect current password",
		"err.focusModeActive":          "focus mode is active — %d min remaining",
		"err.disableDelayActive":       "disable delay active — %.0f hours remaining",
		"err.noDelayConfigured":        "no delay configured",
		"err.invalidDomain":            "invalid domain",
		"err.emptyKeyword":             "empty keyword",
		"err.portRange":                "port must be between 1024 and 65535",
		"err.unsupportedLanguage":      "unsupported language",
		"err.focusDurationRange":       "duration must be between 1 and 1440 minutes",
		"err.prepareUninstallScript":   "could not prepare uninstall script",
		"err.uninstallNeedsAdmin":      "administrator privileges required to complete uninstall",
		"err.prepareInstallScript":     "could not prepare install script",
		"err.caInstallNeedsAdmin":      "administrator access required to install CA certificate",
		"err.caCertNotFound":           "CA certificate not found — enable protection first",
		"err.proxyFailedToStart":       "proxy failed to start",
		"err.proxyStartTimeout":        "proxy did not start on port %d within 5 seconds",
		"err.domainHasPath":            "enter a site name only, without a path (for example example.com)",
		"err.ruleNotFound":             "this site is not in the list",
		"err.ruleNarrowing":            "%s is blocked including its subdomains; to block only the exact site, remove it first (password required)",
		"err.settingsReadOnly":         "the settings file could not be loaded safely, so settings are read-only; the change is active until the app restarts",
		"err.settingsSaveFailed":       "settings could not be saved: %s",
		"err.hostsElevationDeclined":   "administrator permission was declined, so the hosts file was not updated",
		"err.hostsVerifyFailed":        "the hosts file did not contain the expected entries after writing",
		"err.hostsWriteFailed":         "the hosts file could not be updated: %s",
		"err.hostsClearFailed":         "protection is off, but the hosts file entries could not be removed: %s",
		"err.settingsReloadFailed":     "the settings file still could not be read: %s",
		"info.hostsProtectionOff":      "hosts not updated: protection is off",
		"info.hostsSettingsUnreadable": "hosts not updated: the settings file could not be read, so the hosts file was left as it was",
		"notice.ruleBuiltinExempt":     "%s is an essential service that K10 never blocks, so this rule has no effect",

		// ── Defaults stored in config.json ────────────────────────────────
		"config.blockedMessageDefault": "This website has been blocked to help you stay focused and protected.",

		// ── HTTPS/HTTP block page ─────────────────────────────────────────
		"block.pageTitle":    "Blocked — K10 Web Protection",
		"block.headerTitle":  "K10 Web Protection Administration",
		"block.chip":         "Access Blocked",
		"block.heading":      "This website has been blocked",
		"block.siteLabel":    "Site:",
		"block.message":      "This website has been blocked by K10 Web Protection because it may contain adult content, malware, phishing attempts, or other material that violates your configured filtering policy.",
		"block.chipFiltered": "Filtered by K10 Web Protection",
		"block.chipContact":  "Contact your administrator to request access",
		// block.copyright is an HTML fragment by design — do not escape it.
		"block.copyright": "Copyright &copy; 2024&ndash;2026 K10WebProtection &mdash; All Rights Reserved.",

		// -- macOS-only errors ---------------------------------------------
		"err.invalidCertFormat":  "invalid certificate format",
		"err.writeProfileFailed": "failed to write profile",
		"err.openProfileFailed":  "failed to open profile installer",

		// -- macOS configuration profile (.mobileconfig), shown by System Settings --
		"profile.caDisplayName": "K10 Web Protection CA",
		"profile.caDescription": "K10 Web Protection root CA",
		"profile.organization":  "K10 Web Protection",
		"profile.displayName":   "K10 Web Protection Certificate",
		"profile.description":   "Installs the K10 Web Protection CA so HTTPS block pages display correctly in all browsers.",

		// ── System tray ────────────────────────────────────────────────────
		"tray.open": "Open K10 Web Protection",
		"tray.exit": "Exit",
	},

	"he": {
		// ── Application chrome ────────────────────────────────────────────
		"app.title":       "K10 Web Protection",
		"app.productName": "K10 Web Protection",

		// ── app.go errors surfaced as UI toasts ───────────────────────────
		"err.incorrectPassword":        "הסיסמה שגויה",
		"err.incorrectCurrentPassword": "הסיסמה הנוכחית שגויה",
		"err.focusModeActive":          "מצב מיקוד פעיל — נותרו %d דקות",
		"err.disableDelayActive":       "השהיית הביטול פעילה — נותרו %.0f שעות",
		"err.noDelayConfigured":        "לא הוגדרה השהיה",
		"err.invalidDomain":            "האתר אינו תקין",
		"err.emptyKeyword":             "מילת המפתח ריקה",
		"err.portRange":                "מספר הפורט חייב להיות בין 1024 ל-65535",
		"err.unsupportedLanguage":      "השפה אינה נתמכת",
		"err.focusDurationRange":       "משך הזמן חייב להיות בין 1 ל-1440 דקות",
		"err.prepareUninstallScript":   "לא ניתן להכין את סקריפט הסרת ההתקנה",
		"err.uninstallNeedsAdmin":      "נדרשות הרשאות מנהל מערכת כדי להשלים את הסרת ההתקנה",
		"err.prepareInstallScript":     "לא ניתן להכין את סקריפט ההתקנה",
		"err.caInstallNeedsAdmin":      "נדרשת גישת מנהל מערכת כדי להתקין את רשות האישורים (CA)",
		"err.caCertNotFound":           "רשות האישורים (CA) לא נמצאה — יש להפעיל את ההגנה תחילה",
		"err.proxyFailedToStart":       "הפעלת שרת המתווך נכשלה",
		"err.proxyStartTimeout":        "שרת המתווך לא עלה בפורט %d תוך 5 שניות",
		"err.domainHasPath":            "יש להזין שם אתר בלבד, ללא נתיב (לדוגמה example.com)",
		"err.ruleNotFound":             "האתר אינו ברשימה",
		"err.ruleNarrowing":            "\u2068%s\u2069 חסום כולל תת-הדומיינים שלו; כדי לחסום רק את האתר המדויק, יש להסיר אותו תחילה (נדרשת סיסמה)",
		"err.settingsReadOnly":         "לא ניתן היה לטעון את קובץ ההגדרות באופן בטוח, ולכן ההגדרות לקריאה בלבד; השינוי פעיל עד להפעלה מחדש של האפליקציה",
		"err.settingsSaveFailed":       "לא ניתן לשמור את ההגדרות: \u2068%s\u2069",
		"err.hostsElevationDeclined":   "הרשאת מנהל מערכת נדחתה, ולכן קובץ ה-hosts לא עודכן",
		"err.hostsVerifyFailed":        "לאחר הכתיבה, קובץ ה-hosts לא הכיל את הרשומות הצפויות",
		"err.hostsWriteFailed":         "לא ניתן לעדכן את קובץ ה-hosts: \u2068%s\u2069",
		"err.hostsClearFailed":         "ההגנה כבויה, אך לא ניתן היה להסיר את הרשומות מקובץ ה-hosts: \u2068%s\u2069",
		"err.settingsReloadFailed":     "עדיין לא ניתן לקרוא את קובץ ההגדרות: \u2068%s\u2069",
		"info.hostsProtectionOff":      "קובץ ה-hosts לא עודכן: ההגנה כבויה",
		"info.hostsSettingsUnreadable": "קובץ ה-hosts לא עודכן: לא ניתן היה לקרוא את קובץ ההגדרות, ולכן קובץ ה-hosts נשאר כפי שהיה",
		"notice.ruleBuiltinExempt":     "\u2068%s\u2069 הוא שירות חיוני ש-K10 לעולם אינו חוסם, ולכן לכלל זה אין השפעה",

		// ── Defaults stored in config.json ────────────────────────────────
		"config.blockedMessageDefault": "האתר הזה נחסם כדי לשמור על הריכוז וההגנה שלך.",

		// ── HTTPS/HTTP block page ─────────────────────────────────────────
		"block.pageTitle":    "חסום — K10 Web Protection",
		"block.headerTitle":  "ניהול K10 Web Protection",
		"block.chip":         "הגישה נחסמה",
		"block.heading":      "האתר הזה נחסם",
		"block.siteLabel":    "אתר:",
		"block.message":      "האתר הזה נחסם על ידי K10 Web Protection מכיוון שהוא עשוי להכיל תוכן למבוגרים בלבד, תוכנה זדונית, ניסיונות התחזות, או חומר אחר שמפר את מדיניות הסינון שהוגדרה אצלך.",
		"block.chipFiltered": "מסונן על ידי K10 Web Protection",
		"block.chipContact":  "יש לפנות למנהל המערכת כדי לבקש גישה",
		// block.copyright is an HTML fragment by design — do not escape it.
		"block.copyright": "&copy; 2024&ndash;2026 K10WebProtection &mdash; כל הזכויות שמורות.",

		// -- macOS-only errors ---------------------------------------------
		"err.invalidCertFormat":  "פורמט התעודה אינו תקין",
		"err.writeProfileFailed": "כתיבת הפרופיל נכשלה",
		"err.openProfileFailed":  "פתיחת מתקין הפרופיל נכשלה",

		// -- macOS configuration profile (.mobileconfig), shown by System Settings --
		"profile.caDisplayName": "רשות האישורים (CA) של K10 Web Protection",
		"profile.caDescription": "רשות האישורים (CA) השורשית של K10 Web Protection",
		"profile.organization":  "K10 Web Protection",
		"profile.displayName":   "תעודת K10 Web Protection",
		"profile.description":   "מתקין את רשות האישורים (CA) של K10 Web Protection כדי שדפי חסימת HTTPS יוצגו כראוי בכל הדפדפנים.",

		// ── System tray ────────────────────────────────────────────────────
		"tray.open": "פתח את K10 Web Protection",
		"tray.exit": "יציאה",
	},
}
