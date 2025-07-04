#!/bin/bash

# Script para crear Issues en GitHub a partir de FAQs
# Uso: ./create-faq-issues.sh

REPO_OWNER="scopweb"
REPO_NAME="mcp-go-context"
FAQ_DIR="docs/faq"

echo "🚀 Creando Issues FAQ para el repositorio $REPO_OWNER/$REPO_NAME"
echo "=================================================="

# Verificar que gh CLI está instalado
if ! command -v gh &> /dev/null; then
    echo "❌ Error: GitHub CLI (gh) no está instalado"
    echo "   Instalar desde: https://cli.github.com/"
    exit 1
fi

# Verificar autenticación
if ! gh auth status &> /dev/null; then
    echo "❌ Error: No estás autenticado en GitHub CLI"
    echo "   Ejecuta: gh auth login"
    exit 1
fi

# Lista de FAQs a crear como issues
declare -a faqs=(
    "FAQ-01-no-benefits.md:❓ FAQ - No veo beneficios del MCP después de un día de uso:question,help-wanted,documentation,faq"
    "FAQ-02-empty-memory.md:❓ FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?:question,memory,configuration,faq"
    "FAQ-03-correct-usage.md:❓ FAQ - ¿Cómo usar las herramientas del MCP correctamente?:question,documentation,tools,usage,faq"
    "FAQ-04-disconnection.md:❓ FAQ - El MCP se desconecta después de 60 segundos:bug,connection,fixed,stdio,faq"
    "FAQ-05-claude-config.md:❓ FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?:configuration,claude-desktop,setup,faq"
    "FAQ-06-use-cases.md:❓ FAQ - ¿Cuáles son los casos de uso prácticos del MCP?:documentation,use-cases,examples,workflow,faq"
    "FAQ-07-troubleshooting.md:❓ FAQ - Troubleshooting Problemas comunes y soluciones:troubleshooting,debugging,support,faq"
)

# Crear cada issue
for faq_info in "${faqs[@]}"; do
    IFS=':' read -r filename title labels <<< "$faq_info"
    
    echo ""
    echo "📝 Creando issue: $title"
    echo "   Archivo: $filename"
    echo "   Labels: $labels"
    
    # Verificar que el archivo existe
    if [[ ! -f "$FAQ_DIR/$filename" ]]; then
        echo "   ❌ Error: Archivo $FAQ_DIR/$filename no encontrado"
        continue
    fi
    
    # Crear el issue
    if gh issue create \
        --repo "$REPO_OWNER/$REPO_NAME" \
        --title "$title" \
        --body-file "$FAQ_DIR/$filename" \
        --label "$labels"; then
        echo "   ✅ Issue creado exitosamente"
    else
        echo "   ❌ Error creando issue"
    fi
    
    # Pequeña pausa para evitar rate limiting
    sleep 2
done

echo ""
echo "🎉 Proceso completado!"
echo ""
echo "📋 Próximos pasos:"
echo "   1. Ve a https://github.com/$REPO_OWNER/$REPO_NAME/issues"
echo "   2. Fija (pin) los issues más importantes"
echo "   3. Añade label 'pinned' a los básicos"
echo "   4. Cierra issues que sean duplicados"
echo ""
echo "💡 Para actualizar un issue existente:"
echo "   gh issue edit [número] --body-file [archivo]"
