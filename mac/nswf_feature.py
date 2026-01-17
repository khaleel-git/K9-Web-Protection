import time
import os
import subprocess
from nudenet import NudeDetector

# Initialize AI
detector = NudeDetector()

def close_browser_tab():
    print("🚨 18+ Content Detected! Closing tab...")
    script = 'tell application "Google Chrome" to close active tab of front window'
    subprocess.run(["osascript", "-e", script])

def monitor_screen():
    # --- RIGID PATH SETUP ---
    current_folder = os.path.dirname(os.path.abspath(__file__))
    temp_folder = os.path.join(current_folder, "temp_scr")
    
    if not os.path.exists(temp_folder):
        os.makedirs(temp_folder)
    # ------------------------

    print(f"🚀 K10 Blocker Active.")
    try:
        i = 0
        while True:
            # 1. Capture screen to the SPECIFIC path
            # We use the full img_path variable here
            img_path = os.path.join(temp_folder, f"ss_{i}.png")
            result = subprocess.run(["screencapture", "-x", img_path])
            #result = subprocess.run(["screencapture", img_path])

            if result.returncode != 0:
                print("❌ Mac screencapture command failed.")
                time.sleep(2)
                continue

            # 2. Verify file exists
            if not os.path.exists(img_path):
                print(f"❓ File not found at {img_path}")
                time.sleep(1)
                continue

            # 3. Run AI Detection
            try:
                detections = detector.detect(img_path)
                print(f"\nAnalyzing ss_{i}.png...")
                print(f"{detections}")
            except Exception as e:
                print(f"AI Error: {e}")
                continue
            
            # 4. Detection Logic
            unsafe_labels = [
            "FEMALE_GENITALIA_EXPOSED",  # Pussy
            "MALE_GENITALIA_EXPOSED",    # Dick
            "BUTTOCKS_EXPOSED",          # Butt
            "FEMALE_BREAST_EXPOSED",     # Boobs
            "ANUS_EXPOSED",              # Anus
            "FEET_EXPOSED",              # Often used in explicit contexts
            "BELLY_EXPOSED",             # Can indicate partial nudity
            "ARMPITS_EXPOSED"            # Additional anatomical label
            ]
            for d in detections:
                if d['class'] in unsafe_labels and d['score'] > 0.2:
                    print(f"\n🚨 Explicit content detected: ss_{i}.png")
                    print(f"⚠️ {d['class']}")
                    print(f"Score: {d['score']}")
                    close_browser_tab()
                    time.sleep(3)
                    break

            time.sleep(2.5)  # Wait before next capture   
            i += 1

    except KeyboardInterrupt:
        print("\nStopping...")

if __name__ == "__main__":
    monitor_screen()

    