import time
import os
import sys
import json
import subprocess
import pytesseract
import pyautogui
from nudenet import NudeDetector
import win32gui
import win32process
import psutil

# --- CONFIGURATION ---
WORKING_DIR = os.path.dirname(os.path.abspath(__file__))

# UPDATE 1: Windows Tesseract Path (Apne install path ke mutabiq change karein)
pytesseract.pytesseract.tesseract_cmd = r'C:\Tesseract-OCR\tesseract.exe'

# Initialize AI
detector = NudeDetector()

VISUAL_CONFIRMATION_LABELS = [
    "FEMALE_GENITALIA_EXPOSED", "MALE_GENITALIA_EXPOSED", "BUTTOCKS_EXPOSED",
    "FEMALE_BREAST_EXPOSED", "ANUS_EXPOSED", "FEET_EXPOSED", "BELLY_EXPOSED"
]

# UPDATE 2: Windows active app detection
def get_active_app_name():
    try:
        window = win32gui.GetForegroundWindow()
        _, pid = win32process.GetWindowThreadProcessId(window)
        process = psutil.Process(pid)
        return process.name().lower()
    except:
        return "unknown"

# UPDATE 3: Windows window closing method (Alt+F4 send karega)
def close_active_window():
    try:
        # Alt+F4 is the universal Windows shortcut to close a window
        pyautogui.hotkey('alt', 'f4')
        print("🛠 Windows action: Sent Alt+F4 to active window.")
    except Exception as e:
        print(f"⚠️ Failed to close window: {e}")

def load_keywords(filename):
    path = os.path.join(WORKING_DIR, filename)
    try:
        with open(path, 'r') as f:
            data = json.load(f)
            return [str(item).lower() for sublist in data.values() for item in sublist]
    except:
        return []

def resource_path(relative_path):
    try:
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")
    return os.path.join(base_path, relative_path)

def monitor():
    domain_list = load_keywords(resource_path("domains.json"))
    url_list = load_keywords(resource_path("urls.json"))
    multi_list = load_keywords(resource_path("multi-words.json"))

    ai_check_path = os.path.join(WORKING_DIR, "ai_verify.png")

    print(f"🚀 K9 Web Protection Windows Edition Active.")
    print(f"📊 Database Loaded with Domains: {len(domain_list)}, URLs: {len(url_list)}, Multi-Keywords: {len(multi_list)}" )

    try:
        while True:
            try:
                # 1. Take memory snapshot
                try:
                    screen_mem = pyautogui.screenshot()
                except Exception:
                    time.sleep(1)
                    continue
                
                # 2. OCR
                try:
                    text = pytesseract.image_to_string(screen_mem, config='--psm 6').lower()
                except:
                    continue

                # --- HYBRID RULE A: HARD BLOCKS ---
                found_hard = [kw for kw in (domain_list + url_list) if kw in text]
                if found_hard:
                    triggered_keywords = ", ".join(found_hard).replace(".", " (dot) ")
                    print(f"🚨 HARD BLOCK: Triggered by [{triggered_keywords}]")
                    close_active_window()
                    time.sleep(1)
                    continue

                # --- HYBRID RULE B: MULTI-KEYWORDS (AI Verified) ---
                found_suspect = [kw for kw in multi_list if kw in text]
                if found_suspect:
                    screen_mem.save(ai_check_path)
                    try:
                        detections = detector.detect(ai_check_path)
                        if any(d['class'] in VISUAL_CONFIRMATION_LABELS for d in detections):
                            print(f"🚨 AI CONFIRMED: Explicit visual content detected.")
                            triggered_keywords = ", ".join(found_suspect)
                            print(f"🚨 Hybrid BLOCK: Triggered by [{triggered_keywords}]")
                            close_active_window()
                            time.sleep(2)
                    except Exception as ai_err:
                        print(f"⚠️ AI Verification failed: {ai_err}")
                    finally:
                        if os.path.exists(ai_check_path): 
                            os.remove(ai_check_path)

                time.sleep(0.7)

            except Exception as global_err:
                print(f"♻️ System recovered: {global_err}")
                time.sleep(1)

    except KeyboardInterrupt:
        print("🛑 K9 Web Protection Stopped.")

if __name__ == "__main__":
    monitor()