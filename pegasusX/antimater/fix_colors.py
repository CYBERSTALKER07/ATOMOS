with open('app/components/PromptDashboardSection.tsx', 'r') as f:
    content = f.read()

# Replace glow line
content = content.replace('rgba(124,58,237,', 'rgba(16,185,129,')
content = content.replace('rgba(167,139,250,', 'rgba(52,211,153,')

# Replace prompt bg
content = content.replace('rgba(88,28,180,', 'rgba(4,120,87,')
content = content.replace('rgba(49,16,98,', 'rgba(6,78,59,')

with open('app/components/PromptDashboardSection.tsx', 'w') as f:
    f.write(content)

with open('app/components/ask-prompt/AskPromptCard.tsx', 'r') as f:
    content2 = f.read()

content2 = content2.replace('rgba(88,28,180,', 'rgba(4,120,87,')
content2 = content2.replace('rgba(49,16,98,', 'rgba(6,78,59,')

with open('app/components/ask-prompt/AskPromptCard.tsx', 'w') as f:
    f.write(content2)
