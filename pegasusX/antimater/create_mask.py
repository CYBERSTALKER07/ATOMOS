from PIL import Image

# Open the image and convert to RGBA
img = Image.open('/Users/shakhzod/Desktop/V.O.I.D/productionlogo.jpg').convert("RGBA")
datas = img.getdata()

newData = []
for item in datas:
    # item is (R, G, B, A)
    # The image is black and white. 
    # Use the luminance (or just R channel) as the new alpha channel.
    # We want black (0) to be transparent (0 alpha), white (255) to be opaque (255 alpha)
    # Let's set RGB to black and Alpha to the luminance.
    luminance = int(0.299 * item[0] + 0.587 * item[1] + 0.114 * item[2])
    
    # Optional: thresholding to remove artifacts
    # if luminance > 50: alpha = luminance else alpha = 0
    
    # Just use luminance directly for smooth edges
    newData.append((0, 0, 0, luminance))

img.putdata(newData)
img.save('public/productionlogo-mask.png', "PNG")
