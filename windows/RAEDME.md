# K9 Web Protection for Windows (AI-Powered)

A lightweight, AI-powered content monitoring system for Windows that provides real-time screen analysis and automated blocking of inappropriate content.

## Project Description

K9 Web Protection for Windows is an intelligent, next-generation content filtering solution. It represents the modernization of the discontinued K9 software, rebuilt with advanced computer vision and deep learning models to ensure digital safety.

The system uses a **Hybrid Detection Architecture**:

1. **OCR (Tesseract)**: Real-time text extraction from your screen.
2. **NudeDetector (CNN)**: A neural network that verifies if the visual content is actually inappropriate, drastically reducing false positives.

All AI processing occurs **locally on-device**, ensuring complete privacy with zero data transmission to external servers.

## Features

* **Real-time Screen Monitoring**: Analyzes on-screen content across all applications (not just browsers).
* **AI Verification**: Uses NudeDetector to confirm explicit imagery before taking action.
* **Hybrid Blocking**:
* **Hard Blocks**: Instant closure for blacklisted domains/URLs.
* **AI-Verified Blocks**: Contextual analysis for suspicious keywords.


* **Privacy-First**: No data leaves your machine.
* **Persistence**: Designed to run as a background service or scheduled task.

## Requirements

* **Windows 10 or 11**
* **Python 3.12** (Recommended)
* **Tesseract OCR Engine** (Windows Binary)

## Installation

### 1. Install Tesseract OCR

Windows does not have a package manager like Homebrew by default.

1. Download the installer from [UB-Mannheim Tesseract Repo](https://github.com/UB-Mannheim/tesseract/wiki).
2. Install it to the default path: `C:\Program Files\Tesseract-OCR\tesseract.exe`.

### 2. Setup Virtual Environment

Open Git Bash or CMD in the project folder:

```bash
python -m venv venv
# On Windows:
source venv/Scripts/activate

```

### 3. Install Dependencies

```bash
python -m pip install --upgrade pip
pip install -r requirements.txt

```

*(Note: Ensure `pywin32` and `psutil` are installed for Windows process management).*

## Configuration

Ensure the following JSON files are in your project directory:

* `domains.json`: Blocked domains for immediate action.
* `urls.json`: Specific URL patterns to monitor.
* `multi-words.json`: Keywords that trigger an AI visual check.

### Update Tesseract Path

In your script, ensure the Windows path is correctly set:

```python
pytesseract.pytesseract.tesseract_cmd = r'C:\Program Files\Tesseract-OCR\tesseract.exe'

```

## Usage

### Running the Application

```bash
python main.py

```

### Making it "Non-Stoppable" (Windows Task Scheduler)

To ensure K9 starts automatically and restarts if closed:

1. Open **Task Scheduler** and click **Create Task**.
2. **General**: Name it "K9Guard", check "Run with highest privileges".
3. **Triggers**: Set to "At log on".
4. **Actions**: Point to your `K9 Web Protection.exe` or `main.py`.
5. **Settings**: Enable "If the task fails, restart every 1 minute".

## Building Standalone Executable (.exe)

To create a single-file background application:

```bash
pyinstaller --onefile --noconsole --name "K9 Web Protection" \
    --icon "icon.ico" \
    --collect-all nudenet \
    --add-data "domains.json;." \
    --add-data "urls.json;." \
    --add-data "multi-words.json;." \
    main.py

```

*(Note: Use `;` as a separator for `--add-data` on Windows).*

## Privacy & Security

* **Local Execution**: AI models run offline.
* **No Data Logs**: Only real-time console alerts are shown (if console is enabled).
* **Secure File Locking**: Use `icacls` or Windows Security settings to prevent unauthorized deletion of the `.exe`.

## Troubleshooting

* **Access Denied (Error 5)**: Ensure you are running the terminal as **Administrator**.
* **Tesseract Not Found**: Verify the path in the script matches your installation folder.
* **High CPU Usage**: Increase the `time.sleep(0.7)` interval in the main loop to `1.5` or higher.

---

**Version**: 1.1 (Windows Edition)

**Last Updated**: January 2026

**Platform**: Windows 10/11 (x64)