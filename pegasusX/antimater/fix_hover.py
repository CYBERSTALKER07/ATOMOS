import re

with open('app/globals.css', 'r') as f:
    content = f.read()

# Strip out the previous @layer utilities block I added
content = re.sub(r'@layer utilities\s*\{.*?\n\}\n', '', content, flags=re.DOTALL)

# Add standard CSS for the animations
new_css = """
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

/* Bypass Tailwind completely and animate on group hover using standard CSS */
@media (min-width: 768px) {
  .group:hover .custom-timer-icon {
    animation: timer-tick 1.2s ease-in-out infinite;
  }
  .group:hover .custom-radar-icon {
    animation: radar-spin 2s linear infinite;
  }
  .group:hover .custom-rocket-icon {
    animation: rocket-fly 1.2s ease-in-out infinite;
  }
}

.group:hover .custom-float-icon {
  animation: float-icon 2s ease-in-out infinite;
  background-color: rgba(255, 255, 255, 0.1);
  border-color: rgba(255, 255, 255, 0.2);
  color: #6ee7b7; /* emerald-300 */
}
"""

with open('app/globals.css', 'w') as f:
    f.write(content + new_css)

# Patch OurApproach.tsx
with open('app/components/OurApproach.tsx', 'r') as f:
    approach = f.read()

approach = approach.replace('md:group-hover:animate-timer-tick', 'custom-timer-icon')
approach = approach.replace('md:group-hover:animate-radar-spin', 'custom-radar-icon')
approach = approach.replace('md:group-hover:animate-rocket-fly', 'custom-rocket-icon')

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(approach)

# Patch CloudEcosystemBento.tsx
with open('app/components/CloudEcosystemBento.tsx', 'r') as f:
    bento = f.read()

bento = bento.replace('group-hover:animate-float-icon group-hover:bg-white/[0.1] group-hover:border-white/20 group-hover:text-emerald-300', 'custom-float-icon')

with open('app/components/CloudEcosystemBento.tsx', 'w') as f:
    f.write(bento)
