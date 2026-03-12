# GetLink - Grafikarsa Internal Tool

GetLink is a specialized internal tool built with Streamlit to help Grafikarsa administrators generate and export public profile links for students.

## Features

- **Database Query**: Fetch links directly from the Grafikarsa database based on class level (Kelas 10, Kelas 11).
- **Batch Processing**: Upload CSV or Excel files containing NIS numbers to automatically generate profile links.
- **Multi-Format Export**: Export results to TXT, CSV, Word (DOCX), or PDF.

## How to Run

1.  Install dependencies:
    ```bash
    pip install -r requirements.txt
    ```
2.  Run the app:
    ```bash
    streamlit run app.py
    ```

## Design

The app follows Grafikarsa's neutral monochrome design system.
