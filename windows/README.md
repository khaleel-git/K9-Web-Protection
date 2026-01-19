# K9 Web Protection for Windows (AI-Powered)

A lightweight, AI-driven content monitoring system for Windows that provides real-time screen analysis and automated blocking of inappropriate content.

---

## 📖 Project Description

K9 Web Protection is an intelligent filtering solution rebuilt for the modern Windows ecosystem. It leverages a **Hybrid Detection Architecture**:

* **Text Analysis**: Real-time OCR (Tesseract) extracts on-screen text.
* **Visual Verification**: NudeDetector (CNN) verifies visual triggers, distinguishing between text mentions and actual inappropriate imagery.
* **Privacy-First**: All processing happens locally on-device with **zero data transmission** to external servers.

---

## ✨ Features

* **Real-time Screen Monitoring**: Analyzes content across all applications (not just browsers).
* **AI-Powered Verification**: Confirms visual content using NudeDetector AI before taking action.
* **Hybrid Detection**:
* **Hard Blocks**: Instant closure for blacklisted domains and URLs.
* **AI-Verified Mode**: Contextual analysis for suspicious keyword combinations.


* **Self-Healing**: Automatic service recovery via a background watchdog monitoring script.
* **Silent Operation**: Runs completely hidden in the background via VBScript to avoid user interruption.

---

## 🛠 Requirements

* **OS**: Windows 10 or Windows 11 (64-bit)
* **Python**: Version 3.12 (Strictly required)
* **Engine**: Tesseract OCR for Windows
* **Privileges**: Administrator access for installation and Registry configuration.

---

## 🚀 Installation & Setup

### Step 1: Core Dependencies

1. **Python 3.12**: Install from [python.org](https://www.python.org/downloads/). **Ensure "Add Python to PATH" is checked**.
2. **Tesseract OCR**: Download from [UB-Mannheim](https://github.com/UB-Mannheim/tesseract/wiki). Install to `C:\Tesseract-OCR`.

### Step 2: Build the Application

Open **Command Prompt (Admin)** in your project folder and run:

```bash
pip install -r requirements.txt
pyinstaller --onefile --name "K9 Web Protection" --icon "icon.ico" --collect-all nudenet --add-data "domains.json;." --add-data "urls.json;." --add-data "multi-words.json;." main.py

```

### Step 3: Deployment

Create the necessary system directories and move the built files:

```cmd
mkdir "C:\Program Files\K9 Web Protection"
mkdir "C:\Logs"

```

**Files to copy into `C:\Program Files\K9 Web Protection\`:**

1. `K9 Web Protection.exe` (found in the `dist/` folder).
2. `k9.bat` (The watchdog recovery script).
3. `k9-launcher.vbs` (The silent background launcher).

---

## 🔒 Persistence & Auto-Start Setup

To ensure K9 Web Protection starts automatically when Windows boots, choose one of the following methods. **Method 2 is highly recommended for security and persistence.**

### Method 1: Startup Folder (Basic)

This is the simplest method, but the application can be easily disabled by a user via the Task Manager's "Startup" tab.

1. Press `Win + R`, type `shell:startup`, and press Enter.
2. Create a **Shortcut** to `C:\Program Files\K9 Web Protection\k9-launcher.vbs`.
3. Paste the shortcut into the opened Startup folder.

---

### Method 2: Windows Registry (Recommended & Persistent) 🛡️

This method is professional and "hard-to-disable." It does not appear in the standard Task Manager startup list and requires Administrator privileges to modify or remove.

#### **How to Setup:**

Open **Command Prompt (Admin)** and run the following command:

```cmd
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v "K9Protection" /t REG_SZ /d "wscript.exe \"C:\Program Files\K9 Web Protection\k9-launcher.vbs\"" /f

```

#### **Why the Registry Method is Better:**

* **System-Wide Protection**: It applies to every user account on the computer.
* **Stealth Mode**: It is hidden from standard users and cannot be toggled off in the "Startup Apps" settings.
* **Watchdog Integration**: It triggers the VBScript launcher, which runs the `k9.bat` watchdog in the background, ensuring the protection restarts instantly if the process is killed.

---

## ⚙️ Configuration & Maintenance

| Feature | Configuration Detail |
| --- | --- |
| **Check Interval** | Modify `time.sleep(0.7)` in `main.py` and rebuild the `.exe`. |
| **Watchdog Speed** | Edit `timeout /t 5` in `k9.bat` to change how fast it checks for crashes. |
| **Log Location** | Activity and restarts are logged to `C:\Logs\k9-service.log`. |
| **Updating Configs** | Edit the JSON files, rebuild via PyInstaller, and replace the `.exe`. |

---

## 🛠 Troubleshooting

* **Tesseract Not Found**: Ensure the path in `main.py` matches `C:\Tesseract-OCR\tesseract.exe`.
* **Access Denied**: Ensure you are using an **Administrator** terminal for all `reg` or `mkdir` commands.
* **High CPU Usage**: Increase the sleep interval in `main.py` to `1.5` or `2.0` seconds before rebuilding.
* **Manual Kill**: To stop the hidden background service for maintenance, use:
```cmd
taskkill /F /IM "K9 Web Protection.exe" /T

```



---

## ⚠️ Disclaimer

This software is a digital wellness tool and is not foolproof. It should be used as part of a broader strategy for online safety. Local screen monitoring is performed; ensure all users of the system are aware of the monitoring policy.

---

**Version**: 1.1 | **Updated**: January 2026 | **Platform**: Windows x64