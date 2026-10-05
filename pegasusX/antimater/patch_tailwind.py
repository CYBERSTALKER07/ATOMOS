import re

with open('tailwind.config.ts', 'r') as f:
    content = f.read()

# Add keyframes and animation inside theme.extend
keyframes_str = """
      keyframes: {
        'timer-tick': {
          '0%, 100%': { transform: 'rotate(0deg)' },
          '25%': { transform: 'rotate(12deg)' },
          '50%': { transform: 'rotate(0deg)' },
          '75%': { transform: 'rotate(-12deg)' },
        },
        'radar-spin': {
          '0%': { transform: 'rotate(0deg)' },
          '100%': { transform: 'rotate(360deg)' },
        },
        'rocket-fly': {
          '0%': { transform: 'translate(0, 0) scale(1)' },
          '33%': { transform: 'translate(3px, -3px) scale(1.1)' },
          '66%': { transform: 'translate(-1px, 1px) scale(0.95)' },
          '100%': { transform: 'translate(0, 0) scale(1)' },
        },
        'float-icon': {
          '0%, 100%': { transform: 'translateY(0) scale(1.1)' },
          '50%': { transform: 'translateY(-4px) scale(1.1)' },
        },
      },
      animation: {
        'timer-tick': 'timer-tick 1.5s ease-in-out infinite',
        'radar-spin': 'radar-spin 2s linear infinite',
        'rocket-fly': 'rocket-fly 1.5s ease-in-out infinite',
        'float-icon': 'float-icon 2s ease-in-out infinite',
      },
"""

if "keyframes: {" not in content:
    content = content.replace("extend: {", "extend: {" + keyframes_str)

with open('tailwind.config.ts', 'w') as f:
    f.write(content)
