# Proteggo API

A REST API built with Go that provides a secure way to manage and share images by automatically detecting and obscuring faces. Features a hashtag-based organization system and Firebase integration.

## Features

- Automated face detection in images
- Face obscuring capabilities with temporary and permanent options
- Post management with hashtag categorization
- Role-based access control (Admin/User)
- Real-time notifications using Firebase Cloud Messaging
- Asynchronous image processing using Cloud Tasks

## Tech Stack

### Core
- Go 1.22
- Gin Web Framework
- Firebase Admin SDK

### Google Cloud Platform Services
- Cloud Vision API (for face detection)
- Cloud Storage
- Cloud Firestore
- Cloud Tasks
- Cloud Logging

### Image Processing
- Imaging library for Go
- ExifTool integration

## API Endpoints

### Posts
- `GET /api/posts` - Retrieve posts
- `GET /api/posts/byHashTags/:hashTags` - Get posts by hashtags
- `POST /api/posts` - Create new post (Admin)
- `DELETE /api/posts` - Delete post (Admin)

### Images
- `GET /api/images` - Get images
- `POST /api/images` - Upload images (Admin)
- `DELETE /api/images` - Delete images (Admin)
- `DELETE /api/images/deleteTemp` - Clean temporary images
- `DELETE /api/images/deleteUnused` - Clean unused images

### Face Management
- `GET /api/faces/overlay` - Get face overlay data
- `POST /api/faces/overlay/obscured` - Create permanent face obscuring
- `POST /api/faces/overlay/obscured/temp` - Create temporary face obscuring
- `DELETE /api/faces/overlay` - Delete face overlays

### HashTags
- `GET /api/hashTags` - Get all hashtags
- `GET /api/hashTags/topScored` - Get trending hashtags
- `POST /api/hashTags` - Create hashtags (Admin)
- `DELETE /api/hashTags` - Delete hashtags (Admin)

## Security

- Firebase Authentication
- Admin role verification middleware
- Secure image processing pipeline
- Temporary and permanent face obscuring options