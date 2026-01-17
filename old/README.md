# K10 Web Protection (Ex: K9 Web Protection) 🚫

A lightweight **porn blocker** for Windows that uses OCR and keyword/domain filtering to detect and block inappropriate content.

## ✨ Features
- Blocks domains and keywords defined in generated JSON lists.
- Screen monitoring with OCR (via Tesseract).
- Runs silently in the background (no console window).
- Packaged into a standalone `.exe` using **PyInstaller**.

---

## 📦 Requirements

Install dependencies with:

```bash
pip install -r requirements.txt
````

### Dependencies

* [pytesseract](https://pypi.org/project/pytesseract/) – OCR
* [opencv-python](https://pypi.org/project/opencv-python/) – screenshot / image processing
* [pyautogui](https://pypi.org/project/PyAutoGUI/) – screenshots & automation
* [keyboard](https://pypi.org/project/keyboard/) – hotkey monitoring
* [psutil](https://pypi.org/project/psutil/) – process monitoring
* [pyinstaller](https://pypi.org/project/pyinstaller/) – build executable

⚠️ **Tesseract OCR** must also be installed on Windows separately:
👉 [Download Tesseract OCR (UB Mannheim build)](https://github.com/UB-Mannheim/tesseract/wiki)

After installation, make sure `tesseract.exe` (e.g. `C:\Program Files\Tesseract-OCR\tesseract.exe`) is in your PATH.

---

## 🛠️ Usage

### 1. Generate keyword/domain lists

Run the helper script to process your word lists:

```bash
python build_bad_keywords.py
```

This generates files such as:

* `single_bad_keywords_cloud.json`
* `bad_keywords_cloud.json`
* `domains.json`

---

### 2. Run directly (Python)

```bash
python K10_Blocker.py
```

---

### 3. Build EXE with PyInstaller

```bash
pyinstaller --onefile --noconsole K10_Blocker.py
```

* `--onefile` → single portable exe
* `--noconsole` → hides the console window (runs silently)

The `.exe` will be created in the `dist/` folder.

---

### 4. Auto-run on Startup (recommended)

Instead of Windows services, use **Task Scheduler**:

1. Open **Task Scheduler** → *Create Task*
2. Trigger: **At logon**
3. Action: **Start Program** → select your `K10_Blocker.exe`
4. Check **Run with highest privileges**
5. Done ✅ (it will start hidden at every login)

---

## 📂 Project Structure

```
K10-Blocker/
│
├── lists/                  # Raw keyword/domain text files
│
├── build_bad_keywords.py   # Script to build JSON keyword/domain lists
├── K10_Blocker.py          # Main blocker logic
├── requirements.txt        # Python dependencies
└── README.md               # Documentation
```

---

## ⚡ Notes

* Runs in the background with no taskbar icon.
* JSON lists must be regenerated if you update your raw lists.
* Use `tasklist | findstr K10_Blocker.exe` to check if running.

---

## 📜 License

MIT License. Created for educational and personal use only.