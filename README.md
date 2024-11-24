# Fluxxxer

A simple GTK4 desktop application for generating images using the Flux API. Built with Go and GTK4.

![image](https://github.com/user-attachments/assets/0c854cbc-7dff-484c-ba03-406c7f025ffb)

## Features

- Clean, native GTK4 interface
- Generate multiple images from text prompts
- Real-time image generation progress feedback
- Save generated images locally (Not yet implemented)
- Supports 1:1 aspect ratio PNG outputs (TODO more aspect ratios)

## Prerequisites

- Go 1.23.2 or later
- GTK4 development libraries

## Installation

1. Clone the repository:
```bash
git clone https://github.com/marcusziade/fluxxxer.git
cd fluxxxer
```

2. Create a `.env` file in the project root:
```bash
FLUX_API_URL=your_flux_api_endpoint_here
```

3. Install Go dependencies:
```bash
go mod download
```

## Running

```bash
go run .
```

## Building

To build a binary:
```bash
go build -o fluxxxer
```

## Usage

1. Launch the application
2. Enter your prompt in the text field
3. Click "Generate" to create images
4. Use the "Save" button under each generated image to save it locally

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
