import os
import json
import re

# Buckets
domains = {}
urls = {}
multi_words = {}
single_words = {}

# Path to lists
keyword_file_path = "C:\\Users\\khale\\Documents\\K10-Blocker\\lists\\"

# Regex patterns
domain_pattern = re.compile(r"^(https?:\/\/)?([\w\-]+\.)+[\w\-]+$")
url_pattern = re.compile(r"^(https?:\/\/)?([\w\-]+\.)+[\w\-]+\/[\w\-/_.?=&%]*$")

for folder in os.listdir(keyword_file_path):
    print(f"folder: {folder}")
    for file in os.listdir(os.path.join(keyword_file_path, folder)):
        with open(os.path.join(keyword_file_path, folder, file), "r", encoding="utf-8") as read:
            for line in read:
                keyword = line.strip()
                if not keyword:
                    continue  # skip empty lines
                first_letter = keyword[0].lower()

                # ✅ Check if it's a full URL (contains a slash after domain)
                if url_pattern.match(keyword):
                    if first_letter not in urls:
                        urls[first_letter] = set()
                    urls[first_letter].add(keyword)

                # ✅ Check if it's a domain only
                elif domain_pattern.match(keyword):
                    if first_letter not in domains:
                        domains[first_letter] = set()
                    domains[first_letter].add(keyword)

                # ✅ Check if it's multi-word (space inside)
                elif " " in keyword:
                    if first_letter not in multi_words:
                        multi_words[first_letter] = set()
                    multi_words[first_letter].add(keyword)

                # ✅ Otherwise treat as single word
                else:
                    if first_letter not in single_words:
                        single_words[first_letter] = set()
                    single_words[first_letter].add(keyword)

# Convert sets → lists
for d in (domains, urls, multi_words, single_words):
    for k in d:
        d[k] = list(d[k])

# Save to files
with open("domains.json", "w", encoding="utf-8") as f:
    json.dump(domains, f, indent=4)

with open("urls.json", "w", encoding="utf-8") as f:
    json.dump(urls, f, indent=4)

with open("multi-words.json", "w", encoding="utf-8") as f:
    json.dump(multi_words, f, indent=4)

with open("single-words.json", "w", encoding="utf-8") as f:
    json.dump(single_words, f, indent=4)
