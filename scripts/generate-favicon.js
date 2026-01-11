#!/usr/bin/env node

import fs from 'fs';
import path from 'path';
import sharp from 'sharp';

const svgPath = path.join(process.cwd(), 'public/m-logo.svg');
const faviconPath = path.join(process.cwd(), 'public/favicon.ico');
const outputPath = path.join(process.cwd(), 'public/favicon-32x32.png');

async function generateFavicon() {
  try {
    console.log('🎨 Generating PNG favicon from SVG...');

    const svgBuffer = fs.readFileSync(svgPath);

    await sharp(svgBuffer)
      .resize(32, 32, {
        fit: 'cover',
        position: 'center',
      })
      .png({
        quality: 100,
        compressionLevel: 9,
      })
      .toFile(outputPath);

    console.log('✅ Generated 32x32 PNG favicon:', outputPath);

    await sharp(svgBuffer)
      .resize(32, 32, {
        fit: 'cover',
        position: 'center',
      })
      .png({
        quality: 100,
        compressionLevel: 9,
      })
      .toFile(faviconPath);

    console.log('✅ Generated favicon.ico:', faviconPath);
    console.log('\n🎉 Favicon generation complete!');

  } catch (error) {
    console.error('❌ Error generating favicon:', error);
    process.exit(1);
  }
}

generateFavicon();
