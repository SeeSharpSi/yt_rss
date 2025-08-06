# YouTube RSS Feed Viewer

This is a simple web application that allows you to aggregate YouTube channel RSS feeds into a single, clean interface. You can add and remove channels, filter out YouTube Shorts, and lazy-load videos as you scroll.

## Features

*   **Add & Delete Channels:** Easily add channels by their YouTube handle (e.g., `@mkbhd`).
*   **Filter Shorts:** A simple checkbox allows you to hide or show YouTube Shorts in your feed.
*   **Lazy Loading:** Videos are loaded in batches as you scroll down the page.
*   **Modern UI:** A clean, modern interface with a dark theme.

## Getting Started

### Prerequisites

*   [Go](https://golang.org/doc/install) (version 1.21 or later)
*   [templ](https://templ.guide/introduction/installation)

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/SeeSharpSi/yt_rss
    cd yt_rss
    ```

2.  **Install dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Generate templ files:**
    The `templ` command is used to generate Go code from the `.templ` template files.
    ```bash
    templ generate
    ```
    If you don't have `templ` in your PATH, you can run it via `go run`:
    ```bash
    go run github.com/a-h/templ/cmd/templ generate
    ```

### Configuration

The application uses a `.env` file for configuration. A `SESSION_KEY` is required to run the application. This key is used to encrypt user session cookies and should be a random, 32-byte string.

You can generate a key with the following command:
```bash
openssl rand -hex 32
```
Copy the output of this command and paste it into your `.env` file as the value for `SESSION_KEY`.

### Running the Application

1.  **Build the application:**
    ```bash
    go build
    ```

2.  **Run the executable:**
    ```bash
    ./yt_rss2
    ```
    By default, the application will run on a random available port, which will be printed to the console. You can specify a port with the `-port` flag:
    ```bash
    ./yt_rss2 -port 8080
    ```

## Technologies Used

*   **Backend:** [Go](https://golang.org/)
*   **Frontend:** [templ](https://templ.guide/) for templating and [HTMX](https://htmx.org/) for interactivity.
*   **Feed Parsing:** [gofeed](https://github.com/mmcdole/gofeed)
*   **Routing:** [gorilla/mux](https://github.com/gorilla/mux)
