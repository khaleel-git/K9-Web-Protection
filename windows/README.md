# K9 Web Protection for Windows

A lightweight, AI-powered content monitoring system for Windows that provides real-time screen analysis and automated blocking of inappropriate content.

## Project Description

K9 Web Protection is an intelligent content filtering solution that uses AI and machine learning for comprehensive digital safety on Windows. It combines traditional keyword filtering with advanced computer vision (Tesseract OCR + NudeDetector CNN) to perform real-time screen monitoring and contextual analysis of visual content. Unlike conventional filters, K9's AI-driven approach distinguishes between text mentions and actual inappropriate imagery, reducing false positives while providing system-wide protection across all applications. All processing happens locally on-device with zero data transmission to external servers.

## Features

- **Real-time Screen Monitoring**: Continuously analyzes on-screen content using OCR technology
- **AI-Powered Verification**: Uses NudeDetector AI to confirm visual content before blocking
- **Hybrid Detection System**: 
  - **Hard Blocks**: Immediate closure for known domains and URLs
  - **Multi-Keyword Analysis**: AI verification for suspicious keyword combinations
- **Minimal False Positives**: Distinguishes between text mentions and actual visual content
- **Lightweight**: Runs efficiently in the background with configurable check intervals
- **Auto-Restart**: Automatic service recovery with watchdog monitoring
- **Silent Operation**: Runs hidden in background without visible windows

## Requirements

- Windows 10 or Windows 11 (64-bit)
- **Python 3.12** (specific version required)
- Tesseract OCR for Windows
- Administrator privileges (for installation and auto-start setup)

## Installation

### Step 1: Install Python 3.12

Download and install Python 3.12 from [python.org](https://www.python.org/downloads/)

**Important**: During installation, check the box **"Add Python to PATH"**

### Step 2: Install Tesseract OCR

1. Download the Tesseract installer from: [https://github.com/UB-Mannheim/tesseract/wiki](https://github.com/UB-Mannheim/tesseract/wiki)
2. Run the installer and install to `C:\Tesseract-OCR` (recommended default location)
3. If you install to a different location, update the path in `main.py`:

```python
pytesseract.pytesseract.tesseract_cmd = r'C:\Your\Custom\Path\tesseract.exe'
```

### Step 3: Install Python Dependencies

The project includes a `requirements.txt` file with all necessary dependencies and version constraints.

Open **Command Prompt as Administrator** and run:

```bash
pip install -r requirements.txt
```

**Alternative** - Install dependencies manually:

```bash
pip install "numpy<2.0,>=1.24.0" nudenet>=3.4.2 onnxruntime>=1.17.0 pytesseract>=0.3.13 Pillow>=10.2.0 pyautogui>=0.9.54 psutil>=5.9.8 pywin32 pyinstaller>=6.3.0
```

**Note**: NumPy is locked to version 1.x for NudeNet compatibility. Using NumPy 2.0+ will cause errors.

### Step 4: Build the Executable

Navigate to your project folder and run:

```bash
pyinstaller --onefile --name "K9 Web Protection" --icon "icon.ico" --collect-all nudenet --add-data "domains.json;." --add-data "urls.json;." --add-data "multi-words.json;." main.py
```

This will create `K9 Web Protection.exe` in the `dist` folder. The three JSON configuration files (domains.json, urls.json, multi-words.json) are automatically bundled into the executable.

### Step 5: Create Required Folders

Open **Command Prompt as Administrator** and run:

```bash
mkdir "C:\Program Files\K9 Web Protection"
mkdir "C:\Logs"
```

### Step 6: Copy Files to Installation Directory

Manually copy the following 3 files to `C:\Program Files\K9 Web Protection\`:

1. **K9 Web Protection.exe** (from the `dist` folder after building)
2. **k9.bat** (watchdog script)
3. **k9-launcher.vbs** (silent launcher)

Your final directory structure should look like:
```
C:\Program Files\K9 Web Protection\
├── K9 Web Protection.exe
├── k9.bat
└── k9-launcher.vbs

C:\Logs\
└── (empty, for log files)
```

**Note**: The JSON configuration files (domains.json, urls.json, multi-words.json) are already included inside the .exe file and don't need to be copied separately.

### Step 7: Verify Installation

Test the installation by running:

```bash
cd "C:\Program Files\K9 Web Protection"
"K9 Web Protection.exe"
```

You should see the K9 console start with database statistics.

## Usage

### Running the Application Manually

#### Method 1: Direct Execution
Navigate to the installation folder and double-click `K9 Web Protection.exe`

#### Method 2: Using the Batch File
Double-click `k9.bat` or run from Command Prompt:

```bash
cd "C:\Program Files\K9 Web Protection"
k9.bat
```

#### Method 3: Silent Background Mode (Recommended)
Double-click `k9-launcher.vbs` to run K9 silently in the background without any visible window.

### Running at Startup (Recommended)

To have K9 start automatically when Windows boots:

**Option 1: Startup Folder Method**
1. Press `Win + R` to open the Run dialog
2. Type `shell:startup` and press Enter
3. Right-click in the folder → New → Shortcut
4. Browse to `C:\Program Files\K9 Web Protection\k9-launcher.vbs`
5. Name it "K9 Web Protection" and click Finish

**Option 2: Quick Method**
1. Right-click `k9-launcher.vbs` → Send to → Desktop (create shortcut)
2. Move the shortcut to:
   ```
   C:\Users\YourUsername\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup
   ```

Windows Registry ke zariye setup karne ka tareeqa sabse zyada secure hai kyunke ye Task Manager ke simple "Startup" tab mein nazar nahi aata aur asani se disable nahi hota.

Aap apne `README.md` mein ye section shamil kar sakte hain:

---

#### **Method 4: Run via Windows Registry (Advanced & Persistent)**

This method ensures the K9 Protection launcher starts automatically for all users upon login and is difficult to disable via standard Task Manager settings.

### **1. Add to Registry**

Open **Command Prompt (CMD)** or **Git Bash** as **Administrator** and run the following command:

```shell
reg add "HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Run" /v "K9Protection" /t REG_SZ /d "wscript.exe \"C:\Program Files\K9 Web Protection\k9-launcher.vbs\"" /f

```

### **2. Why Use This Method?**

* **System-Wide Protection**: By using `HKEY_LOCAL_MACHINE`, the protection applies to every user on the computer.
* **Stealth Execution**: It uses `wscript.exe` to trigger the VBScript launcher, which starts the watchdog `.bat` file in a completely hidden background mode.
* **Anti-Disable**: Unlike standard startup shortcuts, registry entries under `Local Machine` require Administrator privileges to modify or remove.

### **3. Verification**

To verify that the entry has been successfully added:

1. Press `Win + R`, type `regedit`, and hit Enter.
2. Navigate to: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
3. You should see a string value named **K9Protection** pointing to your `.vbs` launcher.

### Auto-Restart Feature

The included `k9.bat` file provides automatic restart functionality:

- Monitors if K9 is running every 5 seconds
- Automatically restarts if the process terminates
- Logs all activity to `C:\Logs\k9-service.log`

### Stopping the Application

**If running in Command Prompt**: Press `Ctrl + C`

**If running via k9-launcher.vbs**:
1. Open Task Manager (`Ctrl + Shift + Esc`)
2. Find `K9 Web Protection.exe` in the Processes tab
3. Right-click and select "End Task"

**OR**

Run this command in Command Prompt:
```bash
taskkill /F /IM "K9 Web Protection.exe"
```

## How It Works

### Detection System

1. **Screen Capture**: Takes periodic screenshots for analysis (in-memory, no files saved)
2. **OCR Processing**: Extracts text from screen using Tesseract
3. **Keyword Matching**: Checks for flagged domains, URLs, and keywords
4. **AI Verification**: For multi-keyword triggers, uses NudeDetector to confirm visual content
5. **Action**: Automatically sends `Alt+F4` to close windows containing confirmed inappropriate content

### Detection Modes

**Hard Block Mode**: Domains and URLs in `domains.json` and `urls.json` trigger immediate window closure when detected in text.

**AI-Verified Mode**: Keywords in `multi-words.json` require AI confirmation of explicit visual content before blocking. This prevents false positives from text-only documents.

### Windows-Specific Features

- Uses Windows API (`win32gui`, `win32process`) to detect active applications
- Sends `Alt+F4` keyboard shortcut for universal window closing
- Monitors process names via `psutil` for enhanced detection
- Batch file watchdog ensures service continuity

## Configuration

### Adjusting Check Interval

To modify the check interval, you'll need to edit the source code in `main.py` before building:

```python
time.sleep(0.7)  # Check every 0.7 seconds
```

Then rebuild the executable using the PyInstaller command.

### Watchdog Check Interval

Edit `k9.bat` to change how often the watchdog checks if K9 is running:

```batch
timeout /t 5 >nul  # Change 5 to desired seconds
```

### Updating Configuration Files

Since the JSON files are bundled inside the .exe, to update them:

1. Edit `domains.json`, `urls.json`, or `multi-words.json` in your source folder
2. Rebuild the executable using the PyInstaller command
3. Replace the old `K9 Web Protection.exe` in `C:\Program Files\K9 Web Protection\`

### Log File Location

By default, logs are saved to `C:\Logs\k9-service.log`. To change this, edit the path in `k9.bat`:

```batch
echo Service started at %date% %time% >> C:\Your\Custom\Path\k9-service.log
```

## Privacy & Security

- **Local Processing**: All analysis happens on your device
- **No Network Calls**: No data is sent to external servers
- **No Activity Logging**: Only service status is logged (start/restart events)
- **Temporary Files**: AI verification images are immediately deleted after analysis
- **Memory-Only Screenshots**: Screenshots are processed in RAM and never saved to disk (except during AI verification)

## Troubleshooting

### "Tesseract not found" Error

1. Verify Tesseract is installed:
   ```bash
   C:\Tesseract-OCR\tesseract.exe --version
   ```

2. Update the path in `main.py` if installed elsewhere, then rebuild:
   ```python
   pytesseract.pytesseract.tesseract_cmd = r'C:\Path\To\tesseract.exe'
   ```

### PyInstaller Build Errors

If you get "Module not found" errors during build:

```bash
pip install --upgrade pytesseract pyautogui nudenet pillow pywin32 psutil pyinstaller
```

### K9 Keeps Restarting

Check `C:\Logs\k9-service.log` for error messages. Common causes:
- Tesseract not installed or path incorrect
- Missing dependencies
- Corrupted executable - try rebuilding

### High CPU Usage

Edit `main.py` to increase the check interval:
```python
time.sleep(1.5)  # Check every 1.5 seconds
```
Then rebuild the executable.

### Alt+F4 Not Closing Applications

Some applications may ignore Alt+F4. This is expected behavior for certain system-critical applications.

### Permission Issues

Always run `k9.bat` or `k9-launcher.vbs` with Administrator privileges if you encounter permission errors.

## File Descriptions

| File | Purpose |
|------|---------|
| `K9 Web Protection.exe` | Main executable with bundled AI models and configs |
| `k9.bat` | Watchdog script that auto-restarts K9 if it crashes |
| `k9-launcher.vbs` | Silent launcher that runs k9.bat without visible window |
| `domains.json` | Hard-blocked domain list (bundled in .exe) |
| `urls.json` | Hard-blocked URL pattern list (bundled in .exe) |
| `multi-words.json` | Keywords requiring AI verification (bundled in .exe) |

## Differences from Original K9 Web Protection

This modern reimplementation offers:

- AI-powered content verification (not just keyword blocking)
- Real-time screen monitoring (not just browser-based)
- Reduced false positives through visual confirmation
- Works across all applications, not just web browsers
- No dependency on external web services
- Native Windows integration with auto-restart capability

## Uninstallation

To completely remove K9 Web Protection:

1. Stop the running service (Task Manager → End Task)
2. Remove from Startup folder: `shell:startup` → Delete K9 shortcut
3. Delete installation folder: `C:\Program Files\K9 Web Protection\`
4. Delete logs folder: `C:\Logs\` (optional)

## License

This project is provided as-is for personal use. Please use responsibly and in accordance with local laws and regulations.

## Support

For issues, questions, or feature requests:

1. Check that all dependencies are properly installed
2. Verify files are in the correct location
3. Review `C:\Logs\k9-service.log` for error messages
4. Ensure you rebuilt the executable after any code changes

## Disclaimer

This software is designed as a monitoring tool and should be part of a broader approach to digital wellness. It is not foolproof and should not be relied upon as the sole method of content filtering. Always combine technical solutions with communication, education, and personal accountability.

**Important**: This software monitors screen content and may capture sensitive information during normal operation. Ensure you understand the privacy implications and only use on systems where you have authorization.

---

**Version**: 1.1  
**Last Updated**: January 2026  
**Platform**: Windows 10/11 (64-bit)