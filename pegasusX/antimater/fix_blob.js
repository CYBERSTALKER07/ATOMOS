const fs = require('fs');
let svg = fs.readFileSync('public/blob-anim.svg', 'utf8');

// The file currently has:
// mask with #fff blob and #000 eyes
// solid blob path with fill="#000000"
// masked rect with fill="transparent"

// We want:
// solid blob path to be #ffffff (white, to show through the eye holes)
// masked rect to be #000000 (black, to cover the rest of the blob)

svg = svg.replace('fill="#000000"', 'fill="#ffffff"');
svg = svg.replace('fill="transparent"', 'fill="#000000"');

fs.writeFileSync('public/blob-anim.svg', svg);
