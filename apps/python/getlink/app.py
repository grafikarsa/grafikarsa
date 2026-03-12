import streamlit as st
import pandas as pd
from utils import get_db_connection, fetch_user_links, export_to_txt, export_to_docx, export_to_pdf, export_to_csv, process_batch_file
import io

# Page Config
st.set_page_config(
    page_title="Grafikarsa GetLink",
    page_icon="🔗",
    layout="wide"
)

# Custom Styling (Grafikarsa Neutral Black & White)
st.markdown("""
<style>
    @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');
    
    html, body, [data-testid="stAppViewContainer"] {
        font-family: 'Inter', sans-serif;
        background-color: #FFFFFF;
        color: #111111;
    }
    
    .stButton>button {
        background-color: #111111;
        color: #FFFFFF;
        border-radius: 0.625rem;
        border: none;
        padding: 0.5rem 1rem;
        transition: all 0.2s;
    }
    
    .stButton>button:hover {
        background-color: #333333;
        color: #FFFFFF;
        border: none;
    }
    
    .stTextInput>div>div>input {
        border-radius: 0.625rem;
    }
    
    .stSelectbox>div>div>div {
        border-radius: 0.625rem;
    }
    
    h1, h2, h3 {
        color: #111111;
        font-weight: 700;
    }
    
    .stAlert {
        border-radius: 0.625rem;
    }
    
    /* Hide Streamlit branding */
    #MainMenu {visibility: hidden;}
    footer {visibility: hidden;}
</style>
""", unsafe_allow_html=True)

# Application Header
st.title("🔗 Grafikarsa GetLink")
st.subheader("Internal Tool - Generate Profil User Links")

# Sidebar - DB Credentials
st.sidebar.header("🗄️ Database Credentials")
db_host = st.sidebar.text_input("Server IP / Host", value="localhost")
db_port = st.sidebar.text_input("Port", value="5432")
db_user = st.sidebar.text_input("User", value="grafikarsa")
db_password = st.sidebar.text_input("Password", type="password", value="grafikarsa123")
db_name = st.sidebar.text_input("Database Name", value="grafikarsa")

base_url = st.sidebar.text_input("Base App URL", value="http://localhost:3000")

if "db_conn" not in st.session_state:
    st.session_state.db_conn = None

if st.sidebar.button("Cek Koneksi"):
    conn, err = get_db_connection(db_host, db_port, db_user, db_password, db_name)
    if conn:
        st.session_state.db_conn = conn
        st.sidebar.success("✅ Berhasil Terhubung!")
    else:
        st.sidebar.error(f"❌ Gagal Terhubung: {err}")

# Main Content
tab1, tab2 = st.tabs(["🗄️ Database Query", "📄 CSV Batch Process"])

with tab1:
    if st.session_state.db_conn:
        st.write("### Pilih Data yang akan di Get Link")
        
        option = st.radio(
            "Kategori User:",
            ("User Kelas 10", "User Kelas 11", "Semua User Kelas 10 & 11"),
            help="Pilih jenjang kelas yang ingin dihasilkan link profilnya"
        )
        
        # Mapping selection to levels
        levels = []
        if option == "User Kelas 10":
            levels = [10]
        elif option == "User Kelas 11":
            levels = [11]
        else:
            levels = [10, 11]
            
        if st.button("Generate Links"):
            with st.spinner("Fetching data..."):
                data, err = fetch_user_links(st.session_state.db_conn, levels, base_url)
                
                if data:
                    st.session_state.current_data = data
                    st.write(f"Ditemukan **{len(data)}** user.")
                    
                    df = pd.DataFrame(data)
                    st.dataframe(df[['Nama', 'Kelas', 'Link']], use_container_width=True)
                else:
                    if err:
                        st.error(f"Gagal mengambil data: {err}")
                    else:
                        st.warning("Tidak ada data ditemukan untuk kategori ini.")

        # Download Section
        if "current_data" in st.session_state and st.session_state.current_data:
            st.divider()
            st.write("### 📥 Download File")
            
            col1, col2, col3, col4 = st.columns(4)
            
            data = st.session_state.current_data
            
            # TXT
            txt_data = export_to_txt(data)
            col1.download_button(
                label="Download .TXT",
                data=txt_data,
                file_name=f"links_{option.lower().replace(' ', '_')}.txt",
                mime="text/plain"
            )
            
            # CSV
            csv_data = export_to_csv(data)
            col2.download_button(
                label="Download .CSV",
                data=csv_data,
                file_name=f"links_{option.lower().replace(' ', '_')}.csv",
                mime="text/csv"
            )
            
            # DOCX
            docx_data = export_to_docx(data)
            col3.download_button(
                label="Download .DOCX",
                data=docx_data,
                file_name=f"links_{option.lower().replace(' ', '_')}.docx",
                mime="application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            )
            
            # PDF
            pdf_data = export_to_pdf(data)
            col4.download_button(
                label="Download .PDF",
                data=pdf_data,
                file_name=f"links_{option.lower().replace(' ', '_')}.pdf",
                mime="application/pdf"
            )

    else:
        st.info("💡 Silakan masukkan kredensial database di sidebar dan klik 'Cek Koneksi' untuk melanjutkan.")
        st.image("https://img.freepik.com/free-vector/database-connectivity-concept-illustration_114360-5026.jpg", width=300)

with tab2:
    st.write("### Upload File untuk Batch Processing")
    st.write("Upload file CSV atau Excel yang berisi kolom **NIS**. Sistem akan mendeteksi otomatis dan menambahkan kolom **Link Profil**.")
    st.info("💡 Untuk Google Sheets, silakan 'Download as .csv' atau '.xlsx' terlebih dahulu.")
    
    batch_domain = st.text_input("Domain Profil", value="https://grafikarsa.com")
    uploaded_file = st.file_uploader("Pilih file (CSV, XLSX, XLS)", type=["csv", "xlsx", "xls"])
    
    if uploaded_file is not None:
        with st.spinner("Processing file..."):
            df_result, err = process_batch_file(uploaded_file, batch_domain)
            
            if df_result is not None:
                st.success(f"✅ File {uploaded_file.name} berhasil diproses!")
                st.dataframe(df_result, use_container_width=True)
                
                # Download Result
                # Export to CSV by default for compatibility
                output_csv = df_result.to_csv(index=False).encode('utf-8')
                st.download_button(
                    label="📥 Download Hasil (CSV)",
                    data=output_csv,
                    file_name=f"processed_{uploaded_file.name.split('.')[0]}.csv",
                    mime="text/csv"
                )
                
                # Also allow Excel download if input was excel
                if uploaded_file.name.endswith(('.xlsx', '.xls')):
                    output_excel = io.BytesIO()
                    df_result.to_excel(output_excel, index=False)
                    st.download_button(
                        label="📥 Download Hasil (Excel)",
                        data=output_excel.getvalue(),
                        file_name=f"processed_{uploaded_file.name}",
                        mime="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
                    )
            else:
                st.error(f"Gagal memproses file: {err}")

