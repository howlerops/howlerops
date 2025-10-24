# Getting Started with SQL Studio

Welcome to SQL Studio! This guide will help you get up and running quickly.

## Table of Contents

1. [Installation](#installation)
2. [First-Time Setup](#first-time-setup)
3. [Connecting Your First Database](#connecting-your-first-database)
4. [Writing Your First Query](#writing-your-first-query)
5. [Saving and Organizing Queries](#saving-and-organizing-queries)
6. [Keyboard Shortcuts](#keyboard-shortcuts)

## Installation

### Web Application

Simply visit [app.sqlstudio.com](https://app.sqlstudio.com) and sign up for a free account. No installation required!

### Desktop Applications

#### macOS
1. Download the `.dmg` file from our [downloads page](https://sqlstudio.com/downloads)
2. Open the downloaded file
3. Drag SQL Studio to your Applications folder
4. Launch SQL Studio from Applications

#### Windows
1. Download the `.exe` installer from our [downloads page](https://sqlstudio.com/downloads)
2. Run the installer
3. Follow the installation wizard
4. Launch SQL Studio from the Start menu

#### Linux
1. Download the `.AppImage` file from our [downloads page](https://sqlstudio.com/downloads)
2. Make it executable: `chmod +x SQL-Studio-*.AppImage`
3. Run the application: `./SQL-Studio-*.AppImage`

## First-Time Setup

When you first launch SQL Studio, you'll be guided through an onboarding wizard:

1. **Welcome Screen**: Learn about key features
2. **Profile Setup**: Tell us about your use case and role
3. **First Connection**: Connect to a database (optional)
4. **Quick Tour**: See the main UI components
5. **First Query**: Run a sample query
6. **Choose Your Path**: Decide what to explore next

You can skip any step and return to it later.

## Connecting Your First Database

### SQLite (Easiest)

Perfect for getting started without any setup:

1. Click **Connections** in the sidebar
2. Click **New Connection**
3. Select **SQLite**
4. Choose "Create new database" or "Open existing file"
5. Click **Connect**

### PostgreSQL

1. Click **Connections** in the sidebar
2. Click **New Connection**
3. Select **PostgreSQL**
4. Enter your connection details:
   - **Host**: localhost (or your server address)
   - **Port**: 5432 (default)
   - **Database**: your_database_name
   - **Username**: your_username
   - **Password**: your_password
5. Click **Test Connection**
6. If successful, click **Save**

### MySQL

1. Click **Connections** in the sidebar
2. Click **New Connection**
3. Select **MySQL**
4. Enter your connection details:
   - **Host**: localhost (or your server address)
   - **Port**: 3306 (default)
   - **Database**: your_database_name
   - **Username**: your_username
   - **Password**: your_password
5. Click **Test Connection**
6. If successful, click **Save**

### Connection String Format

You can also use connection strings:

- **PostgreSQL**: `postgresql://user:password@host:5432/database`
- **MySQL**: `mysql://user:password@host:3306/database`
- **SQLite**: `sqlite:///path/to/database.db`

## Writing Your First Query

1. Select a connection from the sidebar
2. Click **New Query** or press `Cmd/Ctrl + N`
3. Type your SQL query in the editor:
   ```sql
   SELECT * FROM users LIMIT 10;
   ```
4. Click **Run** or press `Cmd/Ctrl + Enter`
5. View results in the panel below

### Query Editor Features

- **Syntax Highlighting**: SQL keywords are color-coded
- **Auto-Completion**: Press `Tab` to accept suggestions
- **Error Detection**: Syntax errors are underlined
- **Multiple Queries**: Separate queries with semicolons
- **Run Selection**: Highlight specific query to run only that part

## Saving and Organizing Queries

### Saving a Query

1. Write your query
2. Click **Save** or press `Cmd/Ctrl + S`
3. Enter a descriptive name
4. Optionally choose a folder
5. Click **Save**

### Creating Folders

1. Click **Saved Queries** in the sidebar
2. Click the folder icon
3. Enter folder name
4. Click **Create**

### Favoriting Queries

Click the star icon next to any saved query to mark it as a favorite. Favorites appear at the top of your list.

## Keyboard Shortcuts

### General
- `Cmd/Ctrl + N` - New query
- `Cmd/Ctrl + S` - Save query
- `Cmd/Ctrl + O` - Open saved query
- `Cmd/Ctrl + ,` - Settings
- `?` - Open help panel

### Query Editor
- `Cmd/Ctrl + Enter` - Run query
- `Cmd/Ctrl + /` - Toggle comment
- `Tab` - Accept autocomplete suggestion
- `Cmd/Ctrl + D` - Duplicate line
- `Cmd/Ctrl + F` - Find
- `Cmd/Ctrl + H` - Find and replace

### Navigation
- `Cmd/Ctrl + B` - Toggle sidebar
- `Cmd/Ctrl + K` - Quick search
- `Cmd/Ctrl + Shift + P` - Command palette

## Next Steps

Now that you're set up, explore these features:

- **[Query Templates](FEATURE_GUIDES.md#query-templates)**: Create reusable templates
- **[Team Collaboration](FEATURE_GUIDES.md#team-collaboration)**: Invite team members
- **[Cloud Sync](FEATURE_GUIDES.md#cloud-sync)**: Access from any device
- **[AI Assistant](FEATURE_GUIDES.md#ai-assistant)**: Get help writing queries

## Need Help?

- Press `?` to open the help panel anytime
- Visit our [Community Forum](https://community.sqlstudio.com)
- Contact support at [support@sqlstudio.com](mailto:support@sqlstudio.com)
- Check out [video tutorials](https://sqlstudio.com/videos)

Happy querying! 🚀
