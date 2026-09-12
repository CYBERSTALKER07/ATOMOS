import sys

with open('pegasusX/apps/backend-go/main.go', 'r') as f:
    content = f.read()

content = content.replace(
    '"github.com/pegasusx/pegasusx/apps/backend-go/storageroutes"',
    '"github.com/pegasusx/pegasusx/apps/backend-go/storageroutes"\n\t"github.com/pegasusx/pegasusx/apps/backend-go/taxroutes"'
)

with open('pegasusX/apps/backend-go/main.go', 'w') as f:
    f.write(content)
