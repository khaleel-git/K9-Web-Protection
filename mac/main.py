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
    #domain_list = load_keywords("domains.json")
    #url_list = load_keywords("urls.json")
    #multi_list = load_keywords("multi-words.json")

    domain_list = load_keywords(resource_path("domains.json"))
    url_list = load_keywords(resource_path("urls.json"))
    multi_list = load_keywords(resource_path("multi-words.json"))

    ai_check_path = os.path.join(WORKING_DIR, "ai_verify.png")

    print(f"🚀 K10 Turbo Active.")
    print(f"📊 Database: {len(domain_list)} Domains | {len(url_list)} URLs | {len(multi_list)} Multi-Triggers")

    try:
        while True:
            # 1. Take memory snapshot
            try:
                screen_mem = pyautogui.screenshot()
            except:
                continue
            
            # 2. OCR for initial detection
            text = pytesseract.image_to_string(screen_mem, config='--psm 6').lower()

            # --- HYBRID RULE A: HARD BLOCKS ---
            found_hard = [kw for kw in (domain_list + url_list) if kw in text]
            if found_hard:
                display_name = found_hard[0].split('.')[0]
                close_active_window()
                
                # Verify: Is it a visual webpage or just a text doc?
                #screen_mem.save(ai_check_path)
                #detections = detector.detect(ai_check_path)
                #print(detections)
                
                # If AI finds ANY body parts/exposure, confirm it's a site/media
                #if any(d['class'] in VISUAL_CONFIRMATION_LABELS for d in detections):
                #    print(f"🚨 BLOCK CONFIRMED: Found '{display_name}' with visual content.")
                #    close_active_window()
                #    time.sleep(1.5)
                #else:
                    # It found the word, but the screen looks like a plain document
                #    print(f"ℹ️  Keyword '{display_name}' seen, but AI confirms it's just text. Skipping.")
                
                #if os.path.exists(ai_check_path): os.remove(ai_check_path)
                #continue

            # --- HYBRID RULE B: MULTI-KEYWORDS ---
            found_suspect = [kw for kw in multi_list if kw in text]
            if found_suspect:
                display_suspect = found_suspect[0].split('.')[0]
                
                screen_mem.save(ai_check_path)
                detections = detector.detect(ai_check_path)
                
                # Stricter check for multi-keywords (must be explicit)
                if any(d['class'] in VISUAL_CONFIRMATION_LABELS[:5] for d in detections):
                    print(f"🚨 AI CONFIRMED: Explicit content found for '{display_suspect}'.")
                    close_active_window()
                    time.sleep(2)

                if os.path.exists(ai_check_path): os.remove(ai_check_path)

            time.sleep(0.7) 

    except KeyboardInterrupt:
        print("\n👋 Stopping...")

if __name__ == "__main__":
    monitor()