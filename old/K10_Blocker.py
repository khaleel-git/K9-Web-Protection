import sys
import time
import os
import datetime
import json
import re

# Import all necessary libraries for both screen monitoring and GUI blocking.
import pyautogui
import pytesseract

from PyQt6.QtCore import Qt, QUrl, QTimer, pyqtSignal
from PyQt6.QtWidgets import QApplication, QMainWindow, QPushButton, QVBoxLayout, QWidget
from PyQt6.QtWebEngineWidgets import QWebEngineView

# --- USER'S ORIGINAL CONFIGURATION ---
working_dir = "C:\\Users\\khale\\Documents\\K10-Blocker\\"
# Set the path to the Tesseract executable
pytesseract.pytesseract.tesseract_cmd = r'C:\Tesseract-OCR\tesseract.exe'

# --- Safe websites to ignore ---
SAFE_KEYWORDS = {"chatgpt", "linkedin", "meet.com"}

# --- PyQt6 GUI SETUP FOR SCREEN BLOCKING ---

class WebViewWindow(QMainWindow):
    """Full-screen always-on-top web view with unblock button"""
    closed = pyqtSignal()

    def __init__(self, url):
        super().__init__()
        self.showFullScreen()
        self.setWindowFlags(self.windowFlags() | Qt.WindowType.WindowStaysOnTopHint)

        central_widget = QWidget()
        self.setCentralWidget(central_widget)
        layout = QVBoxLayout(central_widget)

        self.browser = QWebEngineView()
        self.browser.setUrl(QUrl(url))
        layout.addWidget(self.browser)

        unblock_button = QPushButton("Unblock Screen")
        unblock_button.setStyleSheet("background-color: #F44336; color: white; padding: 20px; font-size: 24px;")
        unblock_button.clicked.connect(self.close)
        layout.addWidget(unblock_button)

        self.browser.loadFinished.connect(self.show)

    def closeEvent(self, event):
        self.closed.emit()
        super().closeEvent(event)


# --- MAIN APPLICATION LOGIC ---

class ScreenMonitor(QMainWindow):
    """Hidden window that monitors screen using OCR"""
    def __init__(self, single_keywords, multi_keywords, domain_keywords):
        super().__init__()
        
        self.single_Keywords = single_keywords
        self.multi_Keywords = multi_keywords
        self.domain_Keywords = domain_keywords
        self.blocking_window = None
        
        self.setWindowTitle("K10 Blocker Monitor")
        self.hide()
        
        self.timer = QTimer(self)
        self.timer.timeout.connect(self.check_screen)
        self.timer.start(1)  # 1 ms
        
        print(f"K10 Database loaded: {len(single_keywords)} single, {len(multi_keywords)} multi, {len(domain_keywords)} domains")
        print("Screen monitoring started...")

    def get_active_window_title(self):
        """Returns the title of the currently active window"""
        window = win32gui.GetForegroundWindow()
        return win32gui.GetWindowText(window)

    def check_screen(self):
        try:
            screenshot = pyautogui.screenshot()
            text_on_screen = pytesseract.image_to_string(screenshot)
            # with open("page-words.txt", "w", encoding="utf-8") as f:
            #     json.dump(text_on_screen, f, indent=4)

            if any(safe.lower() in text_on_screen.lower() for safe in SAFE_KEYWORDS):
                # print(f"Safe keyword detected on screen → skipping block.")
                return

            # --- Multi-word keywords ---
            for keyword in self.multi_Keywords:
                pattern = r'\b' + re.escape(keyword) + r'\b'
                if re.search(pattern, text_on_screen, re.IGNORECASE):
                    pyautogui.hotkey('ctrl', 'w')  # close active tab
                    self.block_with_reason(keyword, "multi")
                    return

            # # --- Single keywords ---
            # for keyword in self.single_Keywords:
            #     pattern = r'\b' + re.escape(keyword) + r'\b'
            #     if re.search(pattern, text_on_screen, re.IGNORECASE):
            #         # pyautogui.hotkey('ctrl', 'w')  # close active tab
            #         self.block_with_reason(keyword, "single")
            #         return

            # --- Domains ---
            for domain in self.domain_Keywords:
                if domain.lower() in text_on_screen.lower():
                    pyautogui.hotkey('ctrl', 'w')  # close active tab
                    self.block_with_reason(domain, "domain")
                    return
            
            # --- URLs ---
            for url in url_occean:
                if url.lower() in text_on_screen.lower():
                    pyautogui.hotkey('ctrl', 'w')  # close active tab
                    self.block_with_reason(url, "url")
                    return

        except Exception as ex:
            print(f"An error occurred: {ex}")

    def block_with_reason(self, keyword, kind):
        """Helper to block and log reason"""
        window_title = self.get_active_window_title()
        print(f"{kind.title()} keyword '{keyword}' found in window: '{window_title}'")

        with open(working_dir + "hit_keywords.txt", "a", encoding="utf-8") as fd:
            fd.write(f"{kind}:{keyword}, {window_title}, {datetime.datetime.now()} \n")

        self.timer.stop()
        web_page_url = 'https://www.youtube.com/watch?v=iO6jYmuJCuA'
        self.blocking_window = WebViewWindow(web_page_url)
        self.blocking_window.closed.connect(self.resume_monitoring)

    def resume_monitoring(self):
        """Restart monitoring when unblock window closes"""
        self.blocking_window = None
        self.timer.start()
        print("Screen monitoring resumed...")


# --- MAIN EXECUTION ---

if __name__ == '__main__':
    os.environ["QT_ENABLE_HIGHDPI_SCALING"] = "1"
    app = QApplication(sys.argv)
    app.setQuitOnLastWindowClosed(False)

    # Load lists
    # with open(working_dir + "single-words.json", "r", encoding="utf-8") as fd1:
    #     single_keyword_occean = json.load(fd1)
    with open(working_dir + "multi-words.json", "r", encoding="utf-8") as fd2:
        multi_keyword_occean = json.load(fd2)
    with open(working_dir + "domains.json", "r", encoding="utf-8") as fd3:
        domain_occean = json.load(fd3)
    with open(working_dir + "urls.json", "r", encoding="utf-8") as fd4:
        url_occean = json.load(fd4)

    # Flatten dictionaries into lists
    # single_list = [w for v in single_keyword_occean.values() for w in v]
    single_list = []  # No single words used in this version
    multi_list = [w for v in multi_keyword_occean.values() for w in v]
    domain_list = [w for v in domain_occean.values() for w in v]
    url_occean = [w for v in url_occean.values() for w in v]

    monitor = ScreenMonitor(single_list, multi_list, domain_list)
    sys.exit(app.exec())
