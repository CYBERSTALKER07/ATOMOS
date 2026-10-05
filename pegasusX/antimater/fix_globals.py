import re

with open('app/globals.css', 'r') as f:
    content = f.read()

# Remove the previously appended raw classes
content = re.sub(r'@keyframes timer-tick.*', '', content, flags=re.DOTALL)

new_css = """
@layer utilities {
  @keyframes timer-tick {
    0%, 100% { transform: rotate(0deg); }
    25% { transform: rotate(15deg); }
    50% { transform: rotate(0deg); }
    75% { transform: rotate(-15deg); }
  }

  @keyframes radar-spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  @keyframes rocket-fly {
    0% { transform: translate(0, 0) scale(1); }
    33% { transform: translate(4px, -4px) scale(1.15); }
    66% { transform: translate(-2px, 2px) scale(0.95); }
    100% { transform: translate(0, 0) scale(1); }
  }

  @keyframes float-icon {
    0%, 100% { transform: translateY(0) scale(1.15); }
    50% { transform: translateY(-5px) scale(1.15); }
  }

  .animate-timer-tick {
    animation: timer-tick 1.2s ease-in-out infinite;
  }

  .animate-radar-spin {
    animation: radar-spin 2s linear infinite;
  }

  .animate-rocket-fly {
    animation: rocket-fly 1.2s ease-in-out infinite;
  }

  .animate-float-icon {
    animation: float-icon 2s ease-in-out infinite;
  }
}
"""

with open('app/globals.css', 'w') as f:
    f.write(content + new_css)
