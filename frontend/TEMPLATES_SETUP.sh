#!/bin/bash

# Query Templates & Scheduling - Installation Script
# Run this to install required dependencies

echo "Installing required dependencies for Templates feature..."

cd "$(dirname "$0")"

# Install date-fns for date formatting utilities
npm install date-fns

echo ""
echo "✓ Dependencies installed successfully!"
echo ""
echo "Optional: Install react-syntax-highlighter for better SQL highlighting"
echo "  npm install react-syntax-highlighter"
echo "  npm install --save-dev @types/react-syntax-highlighter"
echo ""
echo "The feature currently uses simple <pre><code> blocks for SQL display."
echo "Upgrade to react-syntax-highlighter or CodeMirror for syntax highlighting."
echo ""
echo "Next steps:"
echo "1. Configure API endpoint in .env:"
echo "   NEXT_PUBLIC_API_URL=http://localhost:8080"
echo ""
echo "2. Add routes to your app:"
echo "   - /templates -> TemplatesPage"
echo "   - /schedules -> SchedulesPage"
echo ""
echo "3. Run dev server:"
echo "   npm run dev"
echo ""
echo "See TEMPLATES_IMPLEMENTATION.md for complete integration guide."
