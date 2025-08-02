# Web Interface

## Overview

The web interface provides a modern, intuitive way to interact with the key-value store. It features a clean design with full CRUD operations, real-time updates, and a responsive layout.

## Features

### Core Functionality
- **Add/Update**: Create new key-value pairs or update existing ones
- **Delete**: Remove keys with confirmation dialogs
- **Search/Filter**: Real-time search across keys and values
- **List View**: Display all stored items with pagination-ready design

### User Experience
- **Loading States**: Visual feedback during API operations
- **Error Handling**: User-friendly error messages with auto-dismiss
- **Success Notifications**: Confirmation of successful operations
- **Confirmation Dialogs**: Safe deletion with user confirmation

### Interface Features
- **Responsive Design**: Works on desktop, tablet, and mobile devices
- **Modern UI**: Clean, professional appearance with hover effects
- **Statistics**: Real-time display of total items and storage size
- **Keyboard Shortcuts**: 
  - `Ctrl+K` / `Cmd+K`: Focus search bar
  - `Ctrl+Enter` / `Cmd+Enter`: Submit form when focused on inputs
  - `Escape`: Clear search or cancel edit mode

### Production Ready
- **Edit Mode**: In-place editing of existing items
- **Form Validation**: Client-side validation with error feedback
- **API Error Handling**: Robust error handling with user feedback
- **Accessibility**: Proper labeling and keyboard navigation

## API Integration

The interface integrates with the following REST API endpoints:

- `GET /api/kv/` - List all keys
- `GET /api/kv/{key}` - Get value for a specific key
- `PUT /api/kv/{key}` - Set/update value for a key
- `DELETE /api/kv/{key}` - Delete a key

## Usage

1. Start the key-value store server:
   ```bash
   make run
   ```

2. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

3. Use the interface to:
   - Add new key-value pairs using the form
   - Search existing items using the search bar
   - Edit items by clicking the "Edit" button
   - Delete items by clicking the "Delete" button (with confirmation)

## Technical Details

### Architecture
- **Single Page Application**: All functionality in one HTML file
- **Vanilla JavaScript**: No external dependencies
- **CSS Grid/Flexbox**: Modern responsive layout
- **Fetch API**: Modern HTTP client for API communication

### Browser Compatibility
- Modern browsers with ES6+ support
- Chrome, Firefox, Safari, Edge (latest versions)
- Mobile browsers on iOS and Android

### File Structure
```
web/
├── static/
│   └── index.html    # Complete web interface (HTML + CSS + JS)
└── README.md         # This documentation
```

## Development

The web interface is a single HTML file containing:
- HTML structure with semantic markup
- Embedded CSS with responsive design
- Vanilla JavaScript with modern ES6+ features

To modify the interface, edit `/web/static/index.html` and restart the server.