import re

with open('order/service.go', 'r') as f:
    text = f.read()

# Find each block of `orderStatusChangedEvent{ ... }`
pattern = r'orderStatusChangedEvent\s*\{(.*?)\}'

def repl(match):
    inner = match.group(1)
    
    # Extract values
    type_m = re.search(r'Type:\s*(.+?),', inner)
    ts_m = re.search(r'Timestamp:\s*(.+?),?', inner)
    
    if not type_m or not ts_m:
        return match.group(0) # fallback
        
    type_val = type_m.group(1).strip()
    ts_val = ts_m.group(1).strip()
    
    # Remove Type and Timestamp lines
    inner = re.sub(r'Type:\s*.+?,\n', '', inner)
    inner = re.sub(r'Timestamp:\s*.+?,?\n', '', inner)
    
    base_event = f'\t\t\tBaseEvent: events.BaseEvent{{Type: {type_val}, Timestamp: {ts_val}}},\n'
    
    return 'events.OrderEvent{\n' + base_event + inner + '}'

new_text = re.sub(pattern, repl, text, flags=re.DOTALL)

with open('order/service.go', 'w') as f:
    f.write(new_text)

