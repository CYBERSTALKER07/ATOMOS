from html.parser import HTMLParser

with open('app/components/visuals/PegasusSciFiLogo.tsx', 'r') as f:
    text = f.read()

print("Height prop defaults to:", [line for line in text.split('\n') if 'height =' in line])
