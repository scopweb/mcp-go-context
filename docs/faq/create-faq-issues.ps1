# Script PowerShell para crear Issues en GitHub a partir de FAQs
# Uso: .\create-faq-issues.ps1

$REPO_OWNER = "scopweb"
$REPO_NAME = "mcp-go-context"
$FAQ_DIR = "docs\faq"

Write-Host "🚀 Creando Issues FAQ para el repositorio $REPO_OWNER/$REPO_NAME" -ForegroundColor Green
Write-Host "==================================================" -ForegroundColor Yellow

# Verificar que gh CLI está instalado
if (-not (Get-Command gh -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Error: GitHub CLI (gh) no está instalado" -ForegroundColor Red
    Write-Host "   Instalar desde: https://cli.github.com/" -ForegroundColor Yellow
    exit 1
}

# Verificar autenticación
$authStatus = gh auth status 2>&1
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Error: No estás autenticado en GitHub CLI" -ForegroundColor Red
    Write-Host "   Ejecuta: gh auth login" -ForegroundColor Yellow
    exit 1
}

# Lista de FAQs a crear como issues
$faqs = @(
    "FAQ-01-no-benefits.md|❓ FAQ - No veo beneficios del MCP después de un día de uso|question,help-wanted,documentation,faq",
    "FAQ-02-empty-memory.md|❓ FAQ - ¿Por qué está vacía mi carpeta .mcp-context/memory?|question,memory,configuration,faq",
    "FAQ-03-correct-usage.md|❓ FAQ - ¿Cómo usar las herramientas del MCP correctamente?|question,documentation,tools,usage,faq",
    "FAQ-04-disconnection.md|❓ FAQ - El MCP se desconecta después de 60 segundos|bug,connection,fixed,stdio,faq",
    "FAQ-05-claude-config.md|❓ FAQ - ¿Cómo configurar correctamente el MCP en Claude Desktop?|configuration,claude-desktop,setup,faq",
    "FAQ-06-use-cases.md|❓ FAQ - ¿Cuáles son los casos de uso prácticos del MCP?|documentation,use-cases,examples,workflow,faq",
    "FAQ-07-troubleshooting.md|❓ FAQ - Troubleshooting Problemas comunes y soluciones|troubleshooting,debugging,support,faq"
)

# Crear cada issue
foreach ($faq_info in $faqs) {
    $parts = $faq_info -split '\|'
    $filename = $parts[0]
    $title = $parts[1]
    $labels = $parts[2]
    
    Write-Host ""
    Write-Host "📝 Creando issue: $title" -ForegroundColor Cyan
    Write-Host "   Archivo: $filename" -ForegroundColor Gray
    Write-Host "   Labels: $labels" -ForegroundColor Gray
    
    # Verificar que el archivo existe
    $filePath = Join-Path $FAQ_DIR $filename
    if (-not (Test-Path $filePath)) {
        Write-Host "   ❌ Error: Archivo $filePath no encontrado" -ForegroundColor Red
        continue
    }
    
    # Crear el issue
    try {
        $result = gh issue create --repo "$REPO_OWNER/$REPO_NAME" --title $title --body-file $filePath --label $labels 2>&1
        if ($LASTEXITCODE -eq 0) {
            Write-Host "   ✅ Issue creado exitosamente" -ForegroundColor Green
            Write-Host "   🔗 URL: $result" -ForegroundColor Blue
        } else {
            Write-Host "   ❌ Error creando issue: $result" -ForegroundColor Red
        }
    }
    catch {
        Write-Host "   ❌ Error creando issue: $_" -ForegroundColor Red
    }
    
    # Pequeña pausa para evitar rate limiting
    Start-Sleep -Seconds 2
}

Write-Host ""
Write-Host "🎉 Proceso completado!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Próximos pasos:" -ForegroundColor Yellow
Write-Host "   1. Ve a https://github.com/$REPO_OWNER/$REPO_NAME/issues"
Write-Host "   2. Fija (pin) los issues más importantes"
Write-Host "   3. Añade label 'pinned' a los básicos"
Write-Host "   4. Cierra issues que sean duplicados"
Write-Host ""
Write-Host "💡 Para actualizar un issue existente:" -ForegroundColor Cyan
Write-Host "   gh issue edit [número] --body-file [archivo]"

# Comando adicional para mostrar todos los issues creados
Write-Host ""
Write-Host "📋 Ver todos los issues FAQ creados:" -ForegroundColor Magenta
Write-Host "   gh issue list --repo $REPO_OWNER/$REPO_NAME --label faq"
