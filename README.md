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
├── lists/                  # Global master lists
│   ├── Keywords            # Flagged words for AI verification
│   └── Urls                # Direct domain hard-blocks
├── windows/                # Windows Source & Persistent Assets
│   ├── main.py             # Windows Core Engine
│   ├── k9.bat              # Self-Healing Watchdog script
│   ├── k9-launcher.vbs     # Stealth Background Launcher
│   ├── domains.json        # Windows Hard-block database
│   ├── urls.json           # Windows URL pattern database
│   ├── multi-words.json    # Windows AI-trigger keywords
│   └── README.md           # [Windows Setup Guide]
└── mac/                    # macOS Source & Features
    ├── k9_protection.py    # macOS Core Engine
    ├── domains.json        # macOS Hard-block database
    ├── urls.json           # macOS URL pattern database
    ├── multi-words.json    # macOS AI-trigger keywords
    ├── features/           # OS-specific logic modules
    └── README.md           # [macOS Setup Guide]

```

---

## 🚀 Get Started (Releases)

Ready to deploy? Download the pre-configured binaries and scripts for your OS from the official release page:

📦 **[Download K9 Web Protection V1.1 (Windows & macOS)](https://github.com/khaleel-git/K9-Web-Protection/releases/tag/V1.1)**

---

## 💻 Documentation by Platform

For deep-dive installation steps, including how to lock the software into the **Windows Registry** or **macOS LaunchAgents**, see the guides below:

* 📘 **[Windows Persistent Setup](https://www.google.com/search?q=./windows/README.md)** (Registry locking, Watchdog logic, EXE building)
* 📙 **[macOS Persistent Setup](https://www.google.com/search?q=./mac/README.md)** (Accessibility permissions, LaunchAgents, Homebrew)

---

## 🛠 Core Stack

| Component | Technology |
| --- | --- |
| **Vision** | NudeNet (Convolutional Neural Network) |
| **OCR** | Tesseract Engine (Real-time text extraction) |
| **OS Control** | PyAutoGUI & Win32API (Window Termination) |
| **Persistence** | Windows Registry / macOS LaunchDaemons |

---

## ⚠️ Disclaimer

K9 Web Protection is an aggressive monitoring tool. It is designed to be difficult to uninstall to help users maintain their digital wellness goals. Ensure you have administrative rights and understand that this software monitors on-screen content locally.

**Version**: 1.1 | **Updated**: January 2026 | **Status**: Active & Unstoppable