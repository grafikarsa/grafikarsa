# Python Apps for Grafikarsa

This directory contains Python-based internal tools and applications for the Grafikarsa platform.

## Applications

### 1. [GetLink](./getlink)
An internal tool to generate public profile links for users based on their class level or by batch processing CSV/Excel files.

## Development Setup

To run these apps locally:

1.  Navigate to the specific app directory.
2.  Create a virtual environment:
    ```bash
    python -m venv .venv
    ```
3.  Activate the virtual environment:
    - Windows: `.venv\Scripts\activate`
    - Unix/macOS: `source .venv/bin/activate`
4.  Install dependencies:
    ```bash
    pip install -r requirements.txt
    ```
5.  Run the application (usually via `streamlit run app.py`).
