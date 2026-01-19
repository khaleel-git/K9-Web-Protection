# 🛡️ K9 Web Protection: The Unstoppable AI Porn Blocker

K9 Web Protection is a next-generation, **hard-to-stop** content filtering solution. Rebuilt for the modern era, it replaces static blocklists with **Real-Time Artificial Intelligence** and **Computer Vision** to provide an aggressive shield against inappropriate content across your entire OS.

---

## 🧠 Why K9 is Different (and Harder to Stop)

Traditional blockers are easily bypassed by changing browsers or clearing cookies. **K9 operates at the Kernel/System level:**

* **Visual Intelligence:** It doesn't just look at URLs; it "sees" your screen. If the AI detects explicit imagery via the **NudeDetector CNN**, the window is terminated instantly.
* **The Watchdog Effect:** On Windows, K9 is paired with a hidden watchdog that monitors the process every 5 seconds. If a user tries to "End Task," K9 resurrects itself almost immediately.
* **Stealth Execution:** Runs as a hidden background service. No tray icons, no easy "Quit" buttons, and no visible windows to close.
* **Privacy-First:** Zero data leaves your machine. The AI processing is 100% local.

---

## 📂 Project Structure

```text
K9-Web-Protection/
├── lists/                          # Global master lists
│   ├── Keywords/                   # Flagged words for AI verification
│   └── Urls/                       # Direct domain hard-blocks
├── windows/                        # Windows Source & Persistent Assets
│   ├── main.py                     # Windows Core Engine
│   ├── k9.bat                      # Self-Healing Watchdog script
│   ├── k9-launcher.vbs             # Stealth Background Launcher
│   ├── domains.json                # Windows Hard-block database
│   ├── urls.json                   # Windows URL pattern database
│   ├── multi-words.json            # Windows AI-trigger keywords
│   └── README.md                   # [Windows Setup Guide]
└── mac/                            # macOS Source & Persistent Assets
    ├── main.py                     # macOS Core Engine
    ├── k9_watchdog.sh              # Integrity Enforcer script
    ├── com.k9webprotection.plist   # AI Engine Service
    ├── com.k9webprotection.watchdog.plist # Watchdog Service
    ├── domains.json                # macOS Hard-block database
    ├── urls.json                   # macOS URL pattern database
    ├── multi-words.json            # macOS AI-trigger keywords
    └── README.md                   # [macOS Setup Guide]
```

---

## 🚀 Get Started (Releases)

Ready to deploy? Download the pre-configured binaries and scripts for your OS from the official release page:

📦 **[Download K9 Web Protection V1.1 (Windows & macOS)](https://github.com/khaleel-git/K9-Web-Protection/releases/tag/V1.1)**

---

## 💻 Documentation by Platform

For deep-dive installation steps, including how to lock the software into the **Windows Registry** or **macOS LaunchAgents**, see the guides below:

* 📘 **[Windows Persistent Setup](https://github.com/khaleel-git/K9-Web-Protection/tree/master/windows)** (Registry locking, Watchdog logic, EXE building)
* 📙 **[macOS Persistent Setup](https://github.com/khaleel-git/K9-Web-Protection/tree/master/mac)** (Accessibility permissions, LaunchAgents, Homebrew)

---

## 🛠 Core Stack

K9 Web Protection is built using a high-performance, cross-platform architecture. While the AI logic remains consistent, the system utilizes platform-specific bridges to ensure deep OS integration and hardware acceleration.

| Component | Windows Technology | macOS Technology |
| --- | --- | --- |
| **AI Vision** | NudeNet (CNN) + ONNX Runtime | NudeNet + **TensorFlow Metal (M-Series GPU)** |
| **OCR Engine** | Tesseract OCR | Tesseract OCR (Homebrew) |
| **OS Bridge** | `pywin32` + Win32API | **`pyobjc` (Cocoa/Objective-C Bridge)** |
| **Automation** | PyAutoGUI | PyAutoGUI + **Accessibility API** |
| **Persistence** | Windows Registry (`Run` key) | **LaunchAgents** + **LaunchDaemons** |
| **Process Control** | `psutil` + Batch Watchdog | `psutil` + **Shell Integrity Enforcer** |
| **File Security** | NTFS Permissions | **`chflags uchg` (System Immutability)** |
| **Build Tool** | PyInstaller (.exe) | PyInstaller (.app bundle) |

### 📦 Unified Dependency Logic

To maintain a lightweight footprint, the software dynamically adjusts its requirements based on the host environment:

* **Universal Core**: Both platforms require **NumPy (< 2.0)** for NudeNet compatibility and **Pillow** for real-time image processing.
* **macOS Hardware Acceleration**: On Apple Silicon (M1/M2/M3), K9 utilizes `tensorflow-metal` to run AI inference directly on the GPU, ensuring minimal impact on battery life and CPU performance.
* **Windows Low-Level Hooks**: Uses `pywin32` to interact directly with the Windows Registry and process manager for a tamper-proof installation.

---

## ⚠️ Disclaimer

K9 Web Protection is an aggressive monitoring tool. It is designed to be difficult to uninstall to help users maintain their digital wellness goals. Ensure you have administrative rights and understand that this software monitors on-screen content locally.

**Version**: 1.1 | **Updated**: January 2026 | **Status**: Active & Unstoppable