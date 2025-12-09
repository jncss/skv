#!/usr/bin/env python3
import re

with open('eskv.go', 'r') as f:
    content = f.read()

# 1. Afegir CSS per toasts
toast_css = '.card-hover { transition: all 0.3s ease; }'
		.card-hover:hover { transform: translateY(-2px); box-shadow: 0 10px 25px rgba(0,0,0,0.1); }
		.item-hover { transition: all 0.2s ease; }
		.item-hover:hover { background-color: #eff6ff; transform: translateX(4px); }
		.btn-primary { transition: all 0.2s ease; }
		.btn-primary:hover { transform: scale(1.02); box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3); }
		.stat-card { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
		#toast-container { position: fixed; top: 20px; right: 20px; z-index: 9999; }
		.toast { min-width: 300px; margin-bottom: 10px; padding: 12px 16px; border-radius: 8px; 
			box-shadow: 0 4px 12px rgba(0,0,0,0.15); display: flex; align-items: center; gap: 10px;
			animation: slideIn 0.3s ease-out; color: white; font-size: 14px; }
		.toast.success { background: linear-gradient(135deg, #10b981, #059669); }
		.toast.error { background: linear-gradient(135deg, #ef4444, #dc2626); }
		.toast.warning { background: linear-gradient(135deg, #f59e0b, #d97706); }
		.toast.info { background: linear-gradient(135deg, #3b82f6, #2563eb); }
		@keyframes slideIn { from { transform: translateX(400px); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
		@keyframes slideOut { from { transform: translateX(0); opacity: 1; } to { transform: translateX(400px); opacity: 0; } }'''

content = content.replace(
    '\t\t.stat-card { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }',
    toast_css
)

# 2. Afegir container de toasts al body
content = content.replace(
    '<body class="bg-gradient-to-br from-slate-50 to-slate-100 min-h-screen">',
    '<body class="bg-gradient-to-br from-slate-50 to-slate-100 min-h-screen">\n\t<div id="toast-container"></div>'
)

# 3. Eliminar botó Recuperar
content = re.sub(
    r'<button onclick="showRecoverModal\(\)"[^>]*>.*?Recuperar.*?</button>\s*',
    '',
    content,
    flags=re.DOTALL
)

# 4. Traduir textos principals a anglès
translations = {
    'Gestió professional de bases de dades clau-valor': 'Professional key-value database management',
    'Sense xifrat': 'No encryption',
    'Contrasenya': 'Password',
    'Refrescar': 'Refresh',
    'Nova Base de Dades': 'New Database',
    "Anar a l'arrel del sistema": 'Go to system root',
    'Actualitzar': 'Refresh',
    'Estadístiques': 'Statistics',
    'Compactar': 'Compact',
    'Registres': 'Records',
    'Eliminats': 'Deleted',
    'Mida': 'Size',
    "No s'han trobat registres": 'No records found',
    'Clau': 'Key',
    'Tipus': 'Type',
    'Valor': 'Value',
    'Accions': 'Actions',
    'Editar': 'Edit',
    'Eliminar': 'Delete',
    'Base de Dades Corrupta': 'Corrupted Database',
    'No es poden llegir les dades correctament': 'Cannot read data correctly',
    'Mida del fitxer': 'File size',
    'Recuperar Base de Dades': 'Recover Database',
    'Si us plau, seleccioneu primer una base de dades': 'Please select a database first',
    'Si us plau, afegiu almenys un parell clau-valor': 'Please add at least one key-value pair',
    "Si us plau, introduïu un nom per al backup": 'Please enter a backup name',
    "El nom del fitxer ha d'acabar amb .json": 'Filename must end with .json',
    'Si us plau, seleccioneu un fitxer JSON': 'Please select a JSON file',
    'Si us plau, seleccioneu un fitxer JSON vàlid': 'Please select a valid JSON file',
    "El nom del fitxer ha d'acabar amb .skv": 'Filename must end with .skv',
}

for cat, eng in translations.items():
    content = content.replace(cat, eng)

# 5. Reemplaçar alerts amb showToast
content = content.replace("alert('Please enter a database name')", "showToast('Please enter a database name', 'error')")
content = content.replace("alert('Database name must end with .skv')", "showToast('Database name must end with .skv', 'error')")
content = content.replace("alert('Key cannot be empty')", "showToast('Key cannot be empty', 'error')")
content = content.replace("alert('Please select a database first')", "showToast('Please select a database first', 'warning')")
content = content.replace("alert('Please add at least one key-value pair')", "showToast('Please add at least one key-value pair', 'warning')")
content = content.replace("alert('Please enter a backup name')", "showToast('Please enter a backup name', 'error')")
content = content.replace("alert('Filename must end with .json')", "showToast('Filename must end with .json', 'error')")
content = content.replace("alert('Please select a JSON file')", "showToast('Please select a JSON file', 'error')")
content = content.replace("alert('Please select a valid JSON file')", "showToast('Please select a valid JSON file', 'error')")
content = content.replace("alert('Filename must end with .skv')", "showToast('Filename must end with .skv', 'error')")
content = re.sub(r'alert\(summary\)', "showToast(summary, 'success')", content)

with open('eskv.go', 'w') as f:
    f.write(content)

print("✓ UI updates applied")
