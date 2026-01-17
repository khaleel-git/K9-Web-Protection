# K9 Web Protection for macOS

A lightweight, AI-powered content monitoring system for macOS that provides real-time screen analysis and automated blocking of inappropriate content.

## Description

K9 Web Protection is an intelligent, next-generation content filtering solution that leverages advanced artificial intelligence and machine learning to provide comprehensive digital safety for macOS users. This project represents the continued development and modernization of the discontinued K9 Web Protection software, rebuilt from the ground up with cutting-edge AI technology.

At its core, K9 utilizes a sophisticated hybrid detection architecture combining traditional keyword-based filtering with state-of-the-art computer vision and deep learning models. The system employs Optical Character Recognition (OCR) powered by Tesseract for real-time text extraction, paired with NudeDetector—a convolutional neural network (CNN) trained specifically for image classification and content moderation.

Unlike conventional web filters that rely solely on blocklists and URL filtering, K9's AI-driven approach performs contextual analysis of visual content, distinguishing between benign text references and actual inappropriate imagery. This intelligent verification layer dramatically reduces false positives while maintaining robust protection across all applications system-wide, not just web browsers.

The software implements a two-tier AI decision system: immediate blocking for known threat patterns, and neural network verification for ambiguous content—ensuring both speed and accuracy. Through continuous screen monitoring, natural language processing of on-screen text, and real-time image classification, K9 provides adaptive, intelligent protection that evolves beyond static blocklists.

All AI processing occurs locally on-device, ensuring complete privacy with zero data transmission to external servers. This makes K9 not just a content filter, but a comprehensive AI-powered digital wellness tool designed for the modern macOS ecosystem.

## Features

- **Real-time Screen Monitoring**: Continuously analyzes on-screen content using OCR technology
- **AI-Powered Verification**: Uses NudeDetector AI to confirm visual content before blocking
- **Hybrid Detection System**: 
  - **Hard Blocks**: Immediate closure for known domains and URLs
  - **Multi-Keyword Analysis**: AI verification for suspicious keyword combinations
- **Minimal False Positives**: Distinguishes between text mentions and actual visual content
- **Lightweight**: Runs efficiently in the background with configurable check intervals

## Requirements

- macOS (tested on Apple Silicon and Intel Macs)
- Python 3.8 or higher
- Tesseract OCR installed via Homebrew

## Installation

### 1. Install Homebrew (if not already installed)

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
```

### 2. Install Tesseract OCR

```bash
brew install tesseract
```

### 3. Install Python Dependencies

```bash
pip install pytesseract pyautogui nudenet pillow
```

### 4. Download K9 Web Protection

Clone or download this repository to your desired location.

### 5. Create Configuration Files

Create three JSON files in the same directory as the script:

**domains.json** - List of blocked domains
```json
{
  "blocked": ["example.com", "another-site.com"]
}
```

**urls.json** - List of blocked URL patterns
```json
{
  "blocked": ["specific-page", "blocked-path"]
}
```

**multi-words.json** - Suspicious keyword combinations requiring AI verification
```json
{
  "triggers": ["keyword1", "keyword2", "phrase"]
}
```

## Usage

### Running the Application

```bash
python k9_protection.py
```

### Running at Startup (Optional)

Create a LaunchAgent to run K9 automatically:

1. Create a plist file at `~/Library/LaunchAgents/com.k9.protection.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.k9.protection</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/python3</string>
        <string>/path/to/your/k9_protection.py</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
```

2. Load the LaunchAgent:

```bash
launchctl load ~/Library/LaunchAgents/com.k9.protection.plist
```

### Stopping the Application

Press `Ctrl + C` in the terminal window where K9 is running.

To stop the LaunchAgent:

```bash
launchctl unload ~/Library/LaunchAgents/com.k9.protection.plist
```

## How It Works

### Detection System

1. **Screen Capture**: Takes periodic screenshots for analysis
2. **OCR Processing**: Extracts text from screen using Tesseract
3. **Keyword Matching**: Checks for flagged domains, URLs, and keywords
4. **AI Verification**: For multi-keyword triggers, uses NudeDetector to confirm visual content
5. **Action**: Automatically closes windows containing confirmed inappropriate content

### Detection Modes

**Hard Block Mode**: Domains and URLs in `domains.json` and `urls.json` trigger immediate window closure when detected in text.

**AI-Verified Mode**: Keywords in `multi-words.json` require AI confirmation of explicit visual content before blocking. This prevents false positives from text-only documents.

## Configuration

### Adjusting Check Interval

Modify the `time.sleep()` value at the end of the monitor loop (default: 0.7 seconds):

```python
time.sleep(0.7)  # Check every 0.7 seconds
```

### Tesseract Path

If Tesseract is installed in a different location, update the path:

```python
pytesseract.pytesseract.tesseract_cmd = r'/your/custom/path/to/tesseract'
```

### AI Detection Sensitivity

Modify `VISUAL_CONFIRMATION_LABELS` to adjust what the AI considers as visual content requiring blocking.

## Privacy & Security

- **Local Processing**: All analysis happens on your device
- **No Network Calls**: No data is sent to external servers
- **No Logging**: Activity is not logged to disk (only console output)
- **Temporary Files**: AI verification images are immediately deleted after analysis

## Troubleshooting

### "Tesseract not found" Error

Ensure Tesseract is installed and the path is correct:

```bash
which tesseract
```

Update the `tesseract_cmd` path in the script accordingly.

### Permission Issues

macOS may require accessibility permissions:

1. Go to **System Settings** → **Privacy & Security** → **Accessibility**
2. Add Terminal (or your Python executable) to the allowed apps

### High CPU Usage

Increase the check interval to reduce system load:

```python
time.sleep(1.5)  # Check every 1.5 seconds instead
```

## Building Standalone Application (Optional)

To create a standalone .app bundle using PyInstaller:

```bash
pip install pyinstaller
pyinstaller --onefile --windowed --add-data "domains.json:." --add-data "urls.json:." --add-data "multi-words.json:." k9_protection.py
```

## Differences from Original K9 Web Protection

This modern reimplementation offers:

- AI-powered content verification (not just keyword blocking)
- Real-time screen monitoring (not just browser-based)
- Reduced false positives through visual confirmation
- Works across all applications, not just web browsers
- No dependency on external web services

## License

This project is provided as-is for personal use. Please use responsibly and in accordance with local laws and regulations.

## Support

For issues, questions, or feature requests, please check that all dependencies are properly installed and configurations are correct.

## Disclaimer

This software is designed as a monitoring tool and should be part of a broader approach to digital wellness. It is not foolproof and should not be relied upon as the sole method of content filtering. Always combine technical solutions with communication, education, and personal accountability.

---

**Version**: 1.0  
**Last Updated**: January 2026  
**Platform**: macOS (Apple Silicon & Intel)