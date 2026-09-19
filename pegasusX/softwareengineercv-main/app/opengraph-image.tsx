import { ImageResponse } from 'next/og';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';

export const alt = 'Pegasus — Logistics Operating System';
export const size = { width: 1200, height: 630 };
export const contentType = 'image/png';

export default async function OpenGraphImage() {
  const logoBuffer = await readFile(join(process.cwd(), 'public', 'pegasus.jpg'));
  const logoSrc = `data:image/jpeg;base64,${logoBuffer.toString('base64')}`;

  return new ImageResponse(
    (
      <div
        style={{
          height: '100%',
          width: '100%',
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          backgroundColor: '#000000',
        }}
      >
        <img
          src={logoSrc}
          width={280}
          height={280}
          alt=""
          style={{ objectFit: 'contain' }}
        />
        <p
          style={{
            marginTop: 36,
            fontSize: 56,
            fontWeight: 600,
            color: '#ffffff',
            letterSpacing: '-0.03em',
            lineHeight: 1.1,
          }}
        >
          Pegasus
        </p>
        <p style={{ marginTop: 14, fontSize: 28, color: '#a3a3a3', lineHeight: 1.3 }}>
          Logistics Operating System
        </p>
      </div>
    ),
    { ...size },
  );
}
