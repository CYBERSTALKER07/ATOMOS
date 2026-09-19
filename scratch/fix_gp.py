import re
import sys

with open(sys.argv[1], 'r') as f:
    content = f.read()

def replace_block(text, prefix, replacement):
    # We find the specific pattern
    pattern = r'(if e\.username == "" \|\| e\.password == "" \{\n\s+if !e\.stubMode\(\) \{\n\s+return ExecutionResult\{\}, errGlobalPayUnkeyed\(\)\n\s+\}\n\s+return ExecutionResult\{.*?\}, nil\n\s+\})'
    
    return re.sub(pattern, replacement, text, flags=re.DOTALL)

replacement = """if e.username == "" || e.password == "" {
			return ExecutionResult{}, errGlobalPayUnkeyed()
		}"""

new_content = replace_block(content, "", replacement)

with open(sys.argv[1], 'w') as f:
    f.write(new_content)
