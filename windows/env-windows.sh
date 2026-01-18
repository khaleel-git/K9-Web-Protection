#!/bin/bash

VENV_NAME="venv"

echo "🌐 Creating Windows Virtual Environment: $VENV_NAME..."

# 1. Environment create karein
python -m venv $VENV_NAME

# 2. Windows activation path
echo "🔌 Activating environment..."
source $VENV_NAME/Scripts/activate

# 3. Pip Upgrade (Windows safe method)
echo "📦 Upgrading pip..."
python -m pip install --upgrade pip

# 4. Requirements Installation
if [ -f "requirements.txt" ]; then
    echo "📥 Installing Windows dependencies from requirements.txt..."
    pip install -r requirements.txt
    echo "🚀 Fortress Windows setup complete!"
else
    echo "❌ Error: requirements.txt nahi mili!"
    exit 1
fi

echo "----------------------------------------"
echo "To start working, run:"
echo "source venv/Scripts/activate"