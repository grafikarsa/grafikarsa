import { NextRequest, NextResponse } from 'next/server';

/**
 * Proxy endpoint for uploading files to MinIO
 * This bypasses CORS issues when uploading directly from browser to MinIO
 */
export async function PUT(request: NextRequest) {
  try {
    const presignedUrl = request.headers.get('x-presigned-url');
    const contentType = request.headers.get('content-type');

    console.log('Upload proxy called');

    if (!presignedUrl) {
      return NextResponse.json(
        { error: 'Missing presigned URL' },
        { status: 400 }
      );
    }

    // Get the file body
    const body = await request.arrayBuffer();
    console.log('Body size:', body.byteLength);

    // Use fetch API
    const response = await fetch(presignedUrl, {
      method: 'PUT',
      headers: {
        'Content-Type': contentType || 'application/octet-stream',
        'Content-Length': body.byteLength.toString(),
      },
      body: body,
    });

    console.log('MinIO response:', response.status);

    if (!response.ok) {
      const errorText = await response.text();
      return NextResponse.json(
        { error: 'Upload to storage failed', details: errorText },
        { status: response.status }
      );
    }

    return NextResponse.json({ success: true });
  } catch (error) {
    console.error('Upload proxy error:', error);
    return NextResponse.json(
      { error: 'Internal server error', details: String(error) },
      { status: 500 }
    );
  }
}
