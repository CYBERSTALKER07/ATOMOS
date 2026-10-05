import re

with open('app/components/OurApproach.tsx', 'r') as f:
    content = f.read()

# Replace <Timer size={36} className="text-black" />
# We can add group-hover:animate-timer-tick
content = content.replace(
    '<Timer size={36} className="text-black" />',
    '<Timer size={36} className="text-black transition-all duration-300 md:group-hover:animate-timer-tick" />'
)

content = content.replace(
    '<Radar size={36} className="text-black" />',
    '<Radar size={36} className="text-black transition-all duration-300 md:group-hover:animate-radar-spin" />'
)

content = content.replace(
    '<Rocket size={36} className="text-black" />',
    '<Rocket size={36} className="text-black transition-all duration-300 md:group-hover:animate-rocket-fly" />'
)

# And remove the md:group-hover:scale-110 from their parent div so it doesn't conflict
content = content.replace(
    'md:group-hover:scale-110 transition-transform duration-300',
    ''
)

with open('app/components/OurApproach.tsx', 'w') as f:
    f.write(content)
