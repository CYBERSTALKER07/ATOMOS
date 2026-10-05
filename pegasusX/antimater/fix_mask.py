from PIL import Image

img = Image.open('/Users/shakhzod/Desktop/V.O.I.D/productionlogo.jpg').convert("RGBA")
datas = img.getdata()

newData = []
for item in datas:
    luminance = int(0.299 * item[0] + 0.587 * item[1] + 0.114 * item[2])
    # Threshold at 50 to eliminate JPEG artifacts in the black background
    if luminance < 50:
        alpha = 0
    else:
        # keep it soft above 50, or just make it 255
        alpha = luminance
    newData.append((0, 0, 0, alpha))

img.putdata(newData)
img.save('public/productionlogo-mask.png', "PNG")
