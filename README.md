# Fluxxxer

A modern GTK4 desktop application for generating images using the Flux API. Built with Go and GTK4, featuring a clean, responsive interface.

![image](https://github.com/user-attachments/assets/2f34cc96-cb49-43ab-9490-11e11f2b7cca)

## Features

- Clean, native GTK4 interface with modern controls
- Generate multiple images from text prompts
- Configure aspect ratio and number of outputs
- Real-time image generation progress feedback
- Grid-based image display with proper sizing
- Save generated images locally
- Copy generated images to clipboard
- Multiple aspect ratios support (1:1, 4:3, 3:4, 16:9, 9:16)
- Upscaler feature

## Prerequisites

- Go 1.23.2 or later
- GTK4 development libraries

## Installation

1. Clone the repository:
```bash
git clone https://github.com/guitaripod/fluxxxer.git
cd fluxxxer
```

2. Create a `.env` file in the project root:
```bash
# Required Flux API configuration
FLUX_API_URL=your_flux_api_endpoint_here

# Optional Flux API configuration
FLUX_NUM_OUTPUTS=4           # Default number of images to generate
FLUX_ASPECT_RATIO=1:1        # Default aspect ratio
FLUX_FORMAT=png              # Default output format
FLUX_QUALITY=1               # Default quality setting (1-10)
FLUX_DISABLE_SAFETY=true     # Whether to disable safety checker

# Optional Upscaler API configuration
UPSCALER_API_URL=https://stability-go.fly.dev/api/v1/upscale  # Stability AI upscaler API URL
UPSCALER_API_KEY=your_upscaler_api_key_here                   # Client API key for the upscaler
UPSCALER_APP_ID=your_app_id_here                              # Optional App ID for authentication
UPSCALER_TYPE=fast                                            # Default upscaling type (fast, conservative, creative)

# UI configuration
FLUX_WINDOW_WIDTH=2000       # Initial window width
FLUX_WINDOW_HEIGHT=800       # Initial window height
```

3. Install Go dependencies:
```bash
go mod download
```

## Running

```bash
go run cmd/fluxxxer/main.go
```

## Building

To build a binary:
```bash
go build -o fluxxxer ./cmd/fluxxxer
```

## Environment Configuration

The application will look for the `.env` file in the following locations (in order):

1. Current working directory
2. User's home directory: `~/.fluxxxer/.env`
3. XDG config directory: `~/.config/fluxxxer/.env`
4. Directory containing the executable

## Usage

1. Launch the application
2. Enter your prompt in the text field
3. Adjust generation settings (aspect ratio, number of images)
4. Click "Generate" or press Enter to create images
5. Use the buttons under each generated image to:
   - Save the image locally
   - Copy the image to your clipboard
   - Upscale the image

## Project Structure

```
fluxxxer/
├── cmd/
│   └── fluxxxer/      # Command-line entry point
├── internal/
│   ├── app/           # Application UI and logic
│   ├── config/        # Configuration management
│   ├── flux/          # Flux API client
│   └── upscaler/      # Image upscaling (future)
```

## Development

This project uses:
- [gotk4](https://github.com/diamondburned/gotk4) for GTK4 bindings
- [godotenv](https://github.com/joho/godotenv) for environment variable management

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
