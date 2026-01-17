import time, os,sys
import json
import subprocess
import pytesseract
import pyautogui
from nudenet import NudeDetector

# --- CONFIGURATION ---
WORKING_DIR = os.path.dirname(os.path.abspath(__file__))
pytesseract.pytesseract.tesseract_cmd = r'/opt/homebrew/bin/tesseract'

# Initialize AI
detector = NudeDetector()

# Labels that confirm it's "Visual Content" and not just a text doc
VISUAL_CONFIRMATION_LABELS = [
    #"FACE_FEMALE", "FACE_MALE", # Human faces
    "FEMALE_GENITALIA_EXPOSED", "MALE_GENITALIA_EXPOSED", "BUTTOCKS_EXPOSED", # Exiplicit body parts
    "FEMALE_BREAST_EXPOSED", "ANUS_EXPOSED", "FEET_EXPOSED", "BELLY_EXPOSED"  # Explicit body parts
]

def get_active_app_name():
    script = 'tell application "System Events" to get name of first process whose frontmost is true'
    try:
        return subprocess.check_output(['osascript', '-e', script]).decode().strip().lower()
    except:
        return "unknown"

def load_keywords(filename):
    path = os.path.join(WORKING_DIR, filename)
    try:
        with open(path, 'r') as f:
            data = json.load(f)
            return [str(item).lower() for sublist in data.values() for item in sublist]
    except:
        return []

def close_active_window():
    script = 'tell application "System Events" to tell process (name of first process whose frontmost is true) to keystroke "w" using command down'
    subprocess.run(["osascript", "-e", script])

def resource_path(relative_path):
    """ Get absolute path to resource, works for dev and for PyInstaller """
    try:
        # PyInstaller creates a temp folder and stores path in _MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")
    return os.path.join(base_path, relative_path)

def monitor():
    # Load keywords using the resource_path for PyInstaller compatibility
    domain_list = load_keywords(resource_path("domains.json"))
    url_list = load_keywords(resource_path("urls.json"))
    multi_list = load_keywords(resource_path("multi-words.json"))

    ai_check_path = os.path.join(WORKING_DIR, "ai_verify.png")

    print(f"🚀 K10 Turbo Active.")
    print(f"📊 Database: {len(domain_list)} Domains | {len(url_list)} URLs | {len(multi_list)} Multi-Triggers")

    try:
        while True:
            try:
                # 1. Take memory snapshot
                try:
                    screen_mem = pyautogui.screenshot()
                except Exception as e:
                    # Capture issues (like screen lock) shouldn't kill the app
                    time.sleep(1)
                    continue
                
                # 2. OCR with specific handling for the 0x89 binary error
                try:
                    # config='--psm 6' assumes a single uniform block of text
                    text = pytesseract.image_to_string(screen_mem, config='--psm 6').lower()
                except (UnicodeDecodeError, Exception):
                    # If Tesseract outputs binary garbage or crashes, skip this frame
                    continue

                # --- HYBRID RULE A: HARD BLOCKS (Domains & URLs) ---
                found_hard = [kw for kw in (domain_list + url_list) if kw in text]
                if found_hard:
                    print(f"🚨 HARD BLOCK: Found match in text.")
                    close_active_window()
                    time.sleep(1) # Prevent immediate re-trigger
                    continue

                # --- HYBRID RULE B: MULTI-KEYWORDS (AI Verified) ---
                found_suspect = [kw for kw in multi_list if kw in text]
                if found_suspect:
                    screen_mem.save(ai_check_path)
                    try:
                        detections = detector.detect(ai_check_path)
                        
                        # If AI finds explicit labels, close the window
                        if any(d['class'] in VISUAL_CONFIRMATION_LABELS for d in detections):
                            print(f"🚨 AI CONFIRMED: Explicit visual content detected.")
                            close_active_window()
                            time.sleep(2)
                    except Exception as ai_err:
                        print(f"⚠️ AI Verification failed: {ai_err}")
                    finally:
                        if os.path.exists(ai_check_path): 
                            os.remove(ai_check_path)

                time.sleep(0.7) # CPU-friendly delay

            except Exception as global_err:
                # Catch-all to ensure the 'Fortress' never stays down
                print(f"♻️  System recovered from unexpected error: {global_err}")
                time.sleep(1)

    except KeyboardInterrupt:
        print("🛑 K10 Turbo Stopped by User.")

if __name__ == "__main__":
    monitor()