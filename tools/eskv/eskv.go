package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jncss/skv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	serverURL = "http://localhost:8011/"
	port      = ":8011"
)

var (
	// Heartbeat tracking
	lastHeartbeat  time.Time
	heartbeatMutex sync.RWMutex
	// Database connection pool
	dbPool      = make(map[string]*dbConnection)
	dbPoolMutex sync.RWMutex
	// Current working directory
	baseDir string
)

func init() {
	// Obtenir el directori de treball actual
	var err error
	baseDir, err = os.Getwd()
	if err != nil {
		baseDir = "."
	}
} // dbConnection gestiona l'accés concurrent a una base de dades
type dbConnection struct {
	db       *skv.SKV
	filepath string
	opts     *skv.Options
	mu       sync.Mutex
	refCount int
	lastUsed time.Time
}

// getOrCreateDB obté o crea una connexió a la BD de forma segura
func getOrCreateDB(filepath string, opts *skv.Options) (*dbConnection, error) {
	dbPoolMutex.Lock()
	defer dbPoolMutex.Unlock()

	// Si ja existeix, incrementar refCount
	if conn, exists := dbPool[filepath]; exists {
		conn.refCount++
		conn.lastUsed = time.Now()
		return conn, nil
	}

	// Crear nova connexió
	db, err := skv.OpenWithOptions(filepath, opts)
	if err != nil {
		return nil, err
	}

	conn := &dbConnection{
		db:       db,
		filepath: filepath,
		opts:     opts,
		refCount: 1,
		lastUsed: time.Now(),
	}

	dbPool[filepath] = conn
	return conn, nil
}

// releaseDB allibera una connexió (decrementar refCount)
func releaseDB(filepath string) {
	dbPoolMutex.Lock()
	defer dbPoolMutex.Unlock()

	if conn, exists := dbPool[filepath]; exists {
		conn.refCount--
		conn.lastUsed = time.Now()
	}
}

// withDB executa una funció amb accés exclusiu a la BD
func withDB(filepath string, opts *skv.Options, fn func(*skv.SKV) error) error {
	conn, err := getOrCreateDB(filepath, opts)
	if err != nil {
		return err
	}
	defer releaseDB(filepath)

	// Bloquejar accés exclusiu a aquesta BD
	conn.mu.Lock()
	defer conn.mu.Unlock()

	return fn(conn.db)
}

// cleanupIdleConnections tanca connexions inactives
func cleanupIdleConnections() {
	dbPoolMutex.Lock()
	defer dbPoolMutex.Unlock()

	now := time.Now()
	for filepath, conn := range dbPool {
		// Tancar si no s'ha usat en 5 minuts i no té referències actives
		if conn.refCount == 0 && now.Sub(conn.lastUsed) > 5*time.Minute {
			conn.db.Close()
			delete(dbPool, filepath)
			log.Printf("Closed idle connection: %s", filepath)
		}
	}
}

// closeAllConnections tanca totes les connexions (quan es tanca el servidor)
func closeAllConnections() {
	dbPoolMutex.Lock()
	defer dbPoolMutex.Unlock()

	for filepath, conn := range dbPool {
		conn.db.Close()
		log.Printf("Closed connection: %s", filepath)
	}
	dbPool = make(map[string]*dbConnection)
}

// Funció per obrir la URL en el navegador en mode aplicació (sense barra d'URL)
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		// Intentar Chrome en mode app
		chromePath := findBrowser([]string{
			"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
			"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
		})
		if chromePath != "" {
			cmd = chromePath
			args = []string{"--app=" + url, "--window-size=1400,900"}
		} else {
			// Fallback al navegador per defecte
			cmd = "cmd"
			args = []string{"/c", "start", url}
		}
	case "darwin": // macOS
		// Intentar Chrome/Chromium en mode app
		chromePath := findBrowser([]string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		})
		if chromePath != "" {
			cmd = chromePath
			args = []string{"--app=" + url, "--window-size=1400,900"}
		} else {
			// Fallback a open
			cmd = "open"
			args = []string{url}
		}
	default: // linux, freebsd, openbsd, netbsd
		// Intentar Chrome/Chromium en mode app (incloent Flatpak)
		chromePath := findBrowser([]string{
			"google-chrome",
			"chromium-browser",
			"chromium",
			"flatpak", // S'usarà amb args especials més endavant
		})
		if chromePath != "" {
			// Si és Flatpak, usar comandament especial
			if chromePath == "flatpak" {
				// Intentar Google Chrome Flatpak
				if exec.Command("flatpak", "list", "--app").Run() == nil {
					cmd = "flatpak"
					args = []string{"run", "com.google.Chrome", "--app=" + url, "--window-size=1400,900"}
				} else {
					// Fallback si Flatpak no està disponible
					cmd = "xdg-open"
					args = []string{url}
				}
			} else {
				cmd = chromePath
				args = []string{"--app=" + url, "--window-size=1400,900"}
			}
		} else {
			// Fallback a xdg-open
			cmd = "xdg-open"
			args = []string{url}
		}
	}
	return exec.Command(cmd, args...).Start()
}

// findBrowser cerca un navegador en les rutes especificades
func findBrowser(paths []string) string {
	for _, path := range paths {
		// Per comandaments del sistema, usar LookPath
		if !strings.HasPrefix(path, "/") && !strings.Contains(path, ":\\") {
			if _, err := exec.LookPath(path); err == nil {
				return path
			}
		} else {
			// Per a rutes absolutes, comprovar si el fitxer existeix
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// En Linux, comprovar si Chrome està instal·lat com a Flatpak
	if runtime.GOOS == "linux" {
		if _, err := exec.LookPath("flatpak"); err == nil {
			// Comprovar si Google Chrome està instal·lat com a Flatpak
			cmd := exec.Command("flatpak", "list", "--app", "--columns=application")
			output, err := cmd.Output()
			if err == nil {
				apps := string(output)
				if strings.Contains(apps, "com.google.Chrome") {
					return "flatpak"
				}
			}
		}
	}

	return ""
}

// Pàgina principal
func indexHandler(c echo.Context) error {
	htmlContent := `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>SKV File Manager</title>
	<script src="https://cdn.tailwindcss.com"></script>
	<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
	<style>
		.card-hover { transition: all 0.3s ease; }
		.card-hover:hover { transform: translateY(-2px); box-shadow: 0 10px 25px rgba(0,0,0,0.1); }
		.item-hover { transition: all 0.2s ease; }
		.item-hover:hover { background-color: #eff6ff; transform: translateX(4px); }
		.btn-primary { transition: all 0.2s ease; }
		.btn-primary:hover { transform: scale(1.02); box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3); }
		.stat-card { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); }
		#toast-container { position: fixed; top: 20px; right: 20px; z-index: 9999; pointer-events: none; }
		.toast { min-width: 300px; margin-bottom: 10px; padding: 12px 16px; border-radius: 8px; pointer-events: auto;
			box-shadow: 0 4px 12px rgba(0,0,0,0.15); display: flex; align-items: center; gap: 10px;
			animation: slideIn 0.3s ease-out; color: white; font-size: 14px; font-weight: 500; }
		.toast.success { background: linear-gradient(135deg, #10b981, #059669); }
		.toast.error { background: linear-gradient(135deg, #ef4444, #dc2626); }
		.toast.warning { background: linear-gradient(135deg, #f59e0b, #d97706); }
		.toast.info { background: linear-gradient(135deg, #3b82f6, #2563eb); }
		@keyframes slideIn { from { transform: translateX(400px); opacity: 0; } to { transform: translateX(0); opacity: 1; } }
	</style>
</head>
<body class="bg-gradient-to-br from-slate-50 to-slate-100 min-h-screen">
	<div id="toast-container"></div>
	<div class="container mx-auto px-4 py-3 max-w-7xl">
		<!-- Header -->
		<div class="bg-white rounded-xl shadow-sm border border-slate-200 p-3 mb-3">
			<div class="flex items-center justify-between">
				<div class="flex items-center gap-2">
					<div class="bg-gradient-to-br from-blue-500 to-indigo-600 p-2 rounded-lg">
						<i class="fas fa-database text-2xl text-white"></i>
					</div>
					<div>
						<h1 class="text-2xl font-bold text-slate-800">SKV File Manager</h1>
						<p class="text-sm text-slate-500">key-value database management</p>
					</div>
				</div>
				<div class="flex items-center gap-3">
					<!-- Encryption controls -->
					<div class="flex items-center gap-2 bg-purple-50 border-2 border-purple-200 rounded-lg px-3 py-2">
						<i class="fas fa-lock text-purple-600"></i>
						<select id="encryptionType" onchange="onEncryptionChange()" class="bg-transparent border-none text-sm font-medium text-purple-700 focus:outline-none cursor-pointer">
							<option value="none">No encryption</option>
							<option value="aes">AES-256</option>
							<option value="simplecipher">SimpleCipher</option>
						</select>
					</div>
					<div id="encryptionPasswordField" class="hidden">
						<input type="password" id="encryptionPassword" placeholder="Password" 
							class="px-3 py-2 text-sm border-2 border-amber-200 bg-amber-50 rounded-lg focus:ring-2 focus:ring-amber-500 focus:border-amber-500 outline-none transition">
						<button onclick="refreshWithEncryption()" class="ml-2 px-3 py-2 bg-gradient-to-r from-amber-500 to-orange-500 text-white text-sm font-medium rounded-lg hover:from-amber-600 hover:to-orange-600 shadow-sm transition">
							<i class="fas fa-sync-alt mr-1"></i>Refresh
						</button>
					</div>
					<button onclick="showNewDbModal()" class="btn-primary inline-flex items-center px-4 py-2 bg-gradient-to-r from-green-500 to-emerald-600 text-white text-sm font-medium rounded-lg hover:from-green-600 hover:to-emerald-700 shadow-md">
						<i class="fas fa-plus-circle mr-2"></i>
						New Database
					</button>
				</div>
			</div>
		</div>
		
		<!-- Files List Section -->
		<div class="bg-white rounded-xl shadow-sm border border-slate-200 mb-3 overflow-hidden">
			<div class="bg-gradient-to-r from-slate-50 to-slate-100 px-4 py-2 border-b border-slate-200">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 flex-1">
						<i class="fas fa-folder-open text-blue-500"></i>
						<div id="breadcrumb" class="flex items-center gap-1 text-sm text-slate-700 font-medium"></div>
						<button onclick="navigateTo('/')" class="ml-2 px-2.5 py-1 bg-slate-200 hover:bg-slate-300 rounded text-xs font-semibold text-slate-700 transition" title="Go to system root">
							<i class="fas fa-home mr-1"></i>/
						</button>
					</div>
					<button onclick="refreshFileList()" class="px-3 py-1.5 bg-blue-500 hover:bg-blue-600 text-white rounded-lg text-xs font-medium transition shadow-sm">
						<i class="fas fa-sync-alt mr-1"></i>Refresh
					</button>
				</div>
			</div>
			<div class="p-3">
				<div id="filesList" class="space-y-0.5"></div>
			</div>
		</div>

		<!-- Stats Section -->
		<div id="statsSection" class="hidden bg-white rounded-xl shadow-sm border border-slate-200 p-3 mb-3">
			<div class="flex items-center justify-between mb-2">
				<div class="flex items-center gap-2">
					<i class="fas fa-chart-bar text-indigo-600"></i>
					<h3 class="text-base font-semibold text-slate-800">Statistics</h3>
				</div>
				<div class="flex gap-2">
					<button onclick="showBackupModal()" class="btn-primary inline-flex items-center px-3 py-1.5 bg-gradient-to-r from-green-500 to-emerald-600 text-white text-sm font-medium rounded-lg hover:from-green-600 hover:to-emerald-700 shadow-md">
						<i class="fas fa-download mr-2"></i>
						Backup
					</button>
					<button onclick="showRestoreModal()" class="btn-primary inline-flex items-center px-3 py-1.5 bg-gradient-to-r from-blue-500 to-cyan-600 text-white text-sm font-medium rounded-lg hover:from-blue-600 hover:to-cyan-700 shadow-md">
						<i class="fas fa-upload mr-2"></i>
						Restore
					</button>
					<button onclick="compactDatabase()" class="btn-primary inline-flex items-center px-3 py-1.5 bg-gradient-to-r from-purple-500 to-violet-600 text-white text-sm font-medium rounded-lg hover:from-purple-600 hover:to-violet-700 shadow-md">
						<i class="fas fa-compress-alt mr-2"></i>
						Compact
					</button>
				</div>
			</div>
			<div id="statsGrid" class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-2"></div>
		</div>

		<!-- Records Section -->
		<div id="recordsSection" class="hidden bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden">
			<div class="bg-gradient-to-r from-slate-50 to-slate-100 p-3 border-b border-slate-200">
				<div class="flex items-center gap-2">
					<div class="flex-1 relative">
						<i class="fas fa-search absolute left-3 top-1/2 transform -translate-y-1/2 text-slate-400"></i>
						<input type="text" id="searchInput" placeholder="Search keys..." 
							class="w-full pl-10 pr-4 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none">
					</div>
					<button onclick="showBulkInsertModal()" class="btn-primary inline-flex items-center px-3 py-2 bg-gradient-to-r from-indigo-500 to-purple-600 text-white text-sm font-medium rounded-lg hover:from-indigo-600 hover:to-purple-700 shadow-md">
						<i class="fas fa-layer-group mr-2"></i>
						Bulk Insert
					</button>
					<button onclick="showAddModal()" class="btn-primary inline-flex items-center px-4 py-2 bg-gradient-to-r from-green-500 to-emerald-600 text-white text-sm font-medium rounded-lg hover:from-green-600 hover:to-emerald-700 shadow-md">
						<i class="fas fa-plus-circle mr-2"></i>
						Add Record
					</button>
				</div>
			</div>
			<div id="recordsContent" class="overflow-x-auto"></div>
		</div>
	</div>

	<!-- New Database Modal -->
	<div id="newDbModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-md w-full transform transition-all">
			<div class="bg-gradient-to-r from-green-500 to-emerald-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i class="fas fa-plus-circle text-xl"></i>
						<h3 class="text-lg font-bold">New Database</h3>
					</div>
					<button onclick="closeNewDbModal()" class="text-white hover:text-green-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4">
				<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
					<i class="fas fa-database text-blue-500"></i>
					Filename
				</label>
				<input type="text" id="newDbName" placeholder="exemple.skv" 
					class="w-full px-4 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 outline-none transition">
				<p class="mt-1.5 text-xs text-slate-500"><i class="fas fa-info-circle mr-1"></i>Filename must end with .skv</p>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeNewDbModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="createNewDb()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-green-500 to-emerald-600 rounded-lg hover:from-green-600 hover:to-emerald-700 shadow-md transition">
					<i class="fas fa-check mr-2"></i>Create
				</button>
			</div>
		</div>
	</div>

	<!-- Add/Edit Modal -->
	<div id="editModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto transform transition-all">
			<div class="bg-gradient-to-r from-blue-500 to-indigo-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i id="modalIcon" class="fas fa-edit text-xl"></i>
						<h3 id="modalTitle" class="text-lg font-bold">Edit Record</h3>
					</div>
					<button onclick="closeEditModal()" class="text-white hover:text-blue-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4 space-y-3">
				<div>
					<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
						<i class="fas fa-key text-blue-500"></i>
						Key
					</label>
					<input type="text" id="editKey" 
						class="w-full px-4 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition font-mono">
				</div>
				<div>
					<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
						<i class="fas fa-file-alt text-indigo-500"></i>
						Value
					</label>
					<div class="flex gap-2 mb-2">
						<button id="textModeBtn" onclick="setEditMode('text')" 
							class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg bg-slate-800 text-white shadow-md transition hover:bg-slate-700">
							<i class="fas fa-font mr-1"></i> Text
						</button>
						<button id="hexModeBtn" onclick="setEditMode('hex')" 
							class="flex-1 px-3 py-1.5 text-xs font-semibold rounded-lg bg-slate-200 text-slate-700 transition hover:bg-slate-300">
							<i class="fas fa-hashtag mr-1"></i> Hexadecimal
						</button>
					</div>
					<textarea id="editValue" rows="8" 
						class="w-full px-4 py-2 text-sm font-mono border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition"></textarea>
					<div id="hexError" class="hidden mt-2 text-xs text-red-600 bg-red-50 border-l-4 border-red-500 p-2 rounded-r flex items-center gap-2">
						<i class="fas fa-exclamation-triangle"></i>
						<span id="hexErrorText"></span>
					</div>
				</div>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeEditModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="saveEdit()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-indigo-600 rounded-lg hover:from-blue-600 hover:to-indigo-700 shadow-md transition">
					<i class="fas fa-save mr-2"></i>Save
				</button>
			</div>
		</div>
	</div>

	<!-- Delete Confirmation Modal -->
	<div id="deleteModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-md w-full transform transition-all">
			<div class="bg-gradient-to-r from-red-500 to-rose-600 p-3 rounded-t-2xl">
				<div class="flex items-center gap-2 text-white">
					<i class="fas fa-exclamation-triangle text-xl"></i>
					<h3 class="text-lg font-bold">Confirmar Eliminació</h3>
				</div>
			</div>
			<div class="p-4">
				<div class="flex items-start gap-2 mb-3">
					<div class="flex-shrink-0 w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
						<i class="fas fa-trash-alt text-red-600 text-lg"></i>
					</div>
					<div class="flex-1">
						<p class="text-sm text-slate-700 mb-1.5">Are you sure you want to delete the key:</p>
						<p class="font-mono font-bold text-slate-900 bg-slate-100 px-2 py-1.5 rounded border-l-4 border-red-500">
							<span id="deleteKeyName"></span>
						</p>
					</div>
				</div>
				<div class="bg-amber-50 border-l-4 border-amber-500 p-2 rounded-r">
					<p class="text-xs text-amber-800 flex items-center gap-2">
						<i class="fas fa-info-circle"></i>
						This action cannot be undone
					</p>
				</div>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeDeleteModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="confirmDelete()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-red-500 to-rose-600 rounded-lg hover:from-red-600 hover:to-rose-700 shadow-md transition">
					<i class="fas fa-trash mr-2"></i>Delete
				</button>
			</div>
		</div>
	</div>

	<!-- Bulk Insert Modal -->
	<div id="bulkInsertModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-y-auto transform transition-all">
			<div class="bg-gradient-to-r from-indigo-500 to-purple-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i class="fas fa-layer-group text-xl"></i>
						<h3 class="text-lg font-bold">Bulk Insert Records</h3>
					</div>
					<button onclick="closeBulkInsertModal()" class="text-white hover:text-indigo-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4 space-y-2">
				<div class="bg-blue-50 border-l-4 border-blue-500 p-3 rounded-r">
					<p class="text-sm text-blue-800 flex items-center gap-2">
						<i class="fas fa-info-circle"></i>
						<span>Add key-value pairs. Each line must have the format: <code class="bg-blue-100 px-2 py-1 rounded font-mono">key=value</code></span>
					</p>
				</div>
				<div id="bulkPairs" class="space-y-2">
					<div class="flex items-center gap-2 bulk-pair-row">
						<input type="text" placeholder="Key" class="flex-1 px-3 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-key">
						<span class="text-slate-400 font-bold">=</span>
						<input type="text" placeholder="Value" class="flex-1 px-3 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-value">
						<button onclick="removeBulkPair(this)" class="text-red-500 hover:text-red-700 hover:bg-red-50 p-2 rounded transition" title="Delete">
							<i class="fas fa-trash-alt"></i>
						</button>
					</div>
				</div>
				<button onclick="addBulkPair()" class="w-full py-2 text-sm font-medium text-indigo-600 bg-indigo-50 border-2 border-indigo-200 border-dashed rounded-lg hover:bg-indigo-100 transition">
					<i class="fas fa-plus mr-2"></i>Add more pairs
				</button>
			</div>
			<div class="flex justify-between items-center gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<div class="text-sm text-slate-600">
					<i class="fas fa-lightbulb text-amber-500 mr-1"></i>
					Tip: Press <kbd class="px-1.5 py-0.5 bg-white border border-slate-300 rounded text-xs">Enter</kbd> to quickly add more rows
				</div>
				<div class="flex gap-2">
					<button onclick="closeBulkInsertModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
						<i class="fas fa-times mr-2"></i>Cancel
					</button>
					<button onclick="saveBulkInsert()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-indigo-500 to-purple-600 rounded-lg hover:from-indigo-600 hover:to-purple-700 shadow-md transition">
						<i class="fas fa-save mr-2"></i>Insert All (<span id="bulkCount">1</span>)
					</button>
				</div>
			</div>
		</div>
	</div>

	<!-- Backup Modal -->
	<div id="backupModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-md w-full transform transition-all">
			<div class="bg-gradient-to-r from-green-500 to-emerald-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i class="fas fa-download text-xl"></i>
						<h3 class="text-lg font-bold">Crear Backup</h3>
					</div>
					<button onclick="closeBackupModal()" class="text-white hover:text-green-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4">
				<div class="bg-blue-50 border-l-4 border-blue-500 p-3 rounded-r mb-3">
					<p class="text-sm text-blue-800 flex items-center gap-2">
						<i class="fas fa-info-circle"></i>
						<span>A backup copy will be created in JSON format with all current data.</span>
					</p>
				</div>
				<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
					<i class="fas fa-file-export text-green-500"></i>
					Backup filename
				</label>
				<input type="text" id="backupName" placeholder="backup.json" 
					class="w-full px-4 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500 outline-none transition font-mono">
				<p class="mt-1.5 text-xs text-slate-500"><i class="fas fa-info-circle mr-1"></i>File will be downloaded automatically</p>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeBackupModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="createBackup()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-green-500 to-emerald-600 rounded-lg hover:from-green-600 hover:to-emerald-700 shadow-md transition">
					<i class="fas fa-download mr-2"></i>Create Backup
				</button>
			</div>
		</div>
	</div>

	<!-- Restore Modal -->
	<div id="restoreModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-md w-full transform transition-all">
			<div class="bg-gradient-to-r from-blue-500 to-cyan-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i class="fas fa-upload text-xl"></i>
						<h3 class="text-lg font-bold">Restaurar Backup</h3>
					</div>
					<button onclick="closeRestoreModal()" class="text-white hover:text-blue-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4">
				<div class="bg-amber-50 border-l-4 border-amber-500 p-3 rounded-r mb-3">
					<p class="text-sm text-amber-800 flex items-center gap-2">
						<i class="fas fa-exclamation-triangle"></i>
						<span>Existing keys with the same name will be overwritten. Keys not present in the backup will be maintained.</span>
					</p>
				</div>
				<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
					<i class="fas fa-file-import text-blue-500"></i>
					Select JSON file
				</label>
				<input type="file" id="restoreFile" accept=".json" 
					class="w-full px-4 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 outline-none transition">
				<p class="mt-1.5 text-xs text-slate-500"><i class="fas fa-info-circle mr-1"></i>Format: JSON generated with backup function</p>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeRestoreModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="restoreBackup()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-blue-500 to-cyan-600 rounded-lg hover:from-blue-600 hover:to-cyan-700 shadow-md transition">
					<i class="fas fa-upload mr-2"></i>Restore
				</button>
			</div>
		</div>
	</div>

	<!-- Recover Modal -->
	<div id="recoverModal" class="hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50 backdrop-blur-sm">
		<div class="bg-white rounded-2xl shadow-2xl max-w-md w-full transform transition-all">
			<div class="bg-gradient-to-r from-orange-500 to-red-600 p-3 rounded-t-2xl">
				<div class="flex items-center justify-between">
					<div class="flex items-center gap-2 text-white">
						<i class="fas fa-medkit text-xl"></i>
						<h3 class="text-lg font-bold">Recover Database</h3>
					</div>
					<button onclick="closeRecoverModal()" class="text-white hover:text-orange-100 transition">
						<i class="fas fa-times text-xl"></i>
					</button>
				</div>
			</div>
			<div class="p-4">
				<div class="bg-amber-50 border-l-4 border-amber-500 p-3 mb-4">
					<div class="flex items-start gap-2">
						<i class="fas fa-exclamation-triangle text-amber-600 mt-0.5"></i>
						<div class="text-sm text-amber-800">
							<strong>Warning:</strong> This operation scans the corrupted database byte by byte to recover valid records with correct CRC.
						</div>
					</div>
				</div>
				<label class="block text-sm font-semibold text-slate-700 mb-1.5 flex items-center gap-2">
					<i class="fas fa-database text-orange-500"></i>
					Recovered database name
				</label>
				<input type="text" id="recoverName" placeholder="recuperada.skv" 
					class="w-full px-4 py-2 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-orange-500 focus:border-orange-500 outline-none transition">
				<p class="mt-1.5 text-xs text-slate-500"><i class="fas fa-info-circle mr-1"></i>Current database is considered corrupted and will be attempted to recover</p>
			</div>
			<div class="flex justify-end gap-2 p-3 bg-slate-50 rounded-b-2xl border-t border-slate-200">
				<button onclick="closeRecoverModal()" class="px-4 py-2 text-sm font-medium text-slate-700 bg-white border-2 border-slate-300 rounded-lg hover:bg-slate-100 transition">
					<i class="fas fa-times mr-2"></i>Cancel
				</button>
				<button onclick="recoverDatabase()" class="px-4 py-2 text-sm font-medium text-white bg-gradient-to-r from-orange-500 to-red-600 rounded-lg hover:from-orange-600 hover:to-red-700 shadow-md transition">
					<i class="fas fa-medkit mr-2"></i>Recover
				</button>
			</div>
		</div>
	</div>

	<script>
		let allRecords = [];
		let currentFileName = null;
		let currentPath = '';
		let editMode = 'text';
		let editingIndex = -1;
		let isAddMode = false;
		let deletingKey = null;
		let currentPage = 0;
		const pageSize = 50;

		// Toast notification system
		function showToast(message, type = 'info') {
			const container = document.getElementById('toast-container');
			const toast = document.createElement('div');
			const icons = {
				success: 'fa-check-circle',
				error: 'fa-exclamation-circle',
				warning: 'fa-exclamation-triangle',
				info: 'fa-info-circle'
			};
			
			toast.className = 'toast ' + type;
			toast.innerHTML = '<i class="fas ' + (icons[type] || icons.info) + '"></i><span>' + message + '</span>';
			container.appendChild(toast);
			
			setTimeout(function() {
				toast.style.animation = 'slideOut 0.3s ease-in forwards';
				setTimeout(function() { toast.remove(); }, 300);
			}, 3000);
		}

		// Load file list on page load
		window.addEventListener('DOMContentLoaded', function() {
			refreshFileList();
			
			// Start heartbeat to keep server alive
			setInterval(function() {
				fetch('/api/heartbeat', { method: 'POST' }).catch(function() {
					// Server might be down, ignore errors
				});
			}, 3000); // Send heartbeat every 3 seconds
			
			// Add keyboard support for bulk insert
			document.addEventListener('keydown', function(e) {
				const bulkModal = document.getElementById('bulkInsertModal');
				if (!bulkModal.classList.contains('hidden')) {
					const target = e.target;
					if (e.key === 'Enter' && (target.classList.contains('bulk-key') || target.classList.contains('bulk-value'))) {
						e.preventDefault();
						addBulkPair();
					}
				}
			});
		});

		async function refreshFileList() {
			try {
				const response = await fetch('/api/files?path=' + encodeURIComponent(currentPath));
				if (!response.ok) {
					throw new Error(await response.text());
				}
				const data = await response.json();
				updateBreadcrumb();
				displayFilesList(data);
			} catch (error) {
				showToast('Error loading files: ' + error.message);
			}
		}

		function updateBreadcrumb() {
			const breadcrumb = document.getElementById('breadcrumb');
			
			if (!currentPath || currentPath === '.' || currentPath === '') {
				breadcrumb.innerHTML = '<span class="font-semibold text-slate-800"><i class="fas fa-folder-open mr-1.5 text-blue-500"></i>current directory</span>';
				return;
			}
			
			if (currentPath === '/') {
				breadcrumb.innerHTML = '<span class="font-semibold text-slate-800"><i class="fas fa-server mr-1.5 text-indigo-500"></i>system root</span>';
				return;
			}
			
			const parts = currentPath.split('/').filter(p => p);
			let html = '';
			
			if (currentPath.startsWith('/')) {
				html = '<button onclick="navigateTo(\'/\')" class="hover:text-blue-600 transition"><i class="fas fa-home"></i></button>';
			} else {
				html = '<button onclick="navigateTo(\'\')" class="hover:text-blue-600 transition"><i class="fas fa-dot-circle"></i></button>';
			}
			
			let path = currentPath.startsWith('/') ? '' : '';
			
			parts.forEach((part, idx) => {
				if (currentPath.startsWith('/')) {
					path += '/' + part;
				} else {
					path += (path ? '/' : '') + part;
				}
				html += ' <i class="fas fa-chevron-right text-xs text-slate-400 mx-1"></i> ';
				if (idx === parts.length - 1) {
					html += '<span class="font-semibold text-slate-800">' + part + '</span>';
				} else {
					const p = path;
					html += '<button onclick="navigateTo(\'' + p + '\')" class="hover:text-blue-600 transition">' + part + '</button>';
				}
			});
			
			breadcrumb.innerHTML = html;
		}

		function navigateTo(path) {
			currentPath = path;
			refreshFileList();
		}

		function displayFilesList(data) {
			const container = document.getElementById('filesList');
			if (data.files.length === 0 && data.folders.length === 0) {
				container.innerHTML = '<div class="text-center py-8"><i class="fas fa-inbox text-4xl text-slate-300 mb-3"></i><p class="text-sm text-slate-500">No files or folders found</p><p class="text-xs text-slate-400 mt-1">Create a new database to get started</p></div>';
				return;
			}
			
			let html = '';
			
			// Show folders first
			data.folders.forEach(folder => {
				let newPath;
				if (!currentPath || currentPath === '.' || currentPath === '') {
					newPath = folder.name;
				} else if (currentPath === '/') {
					newPath = '/' + folder.name;
				} else {
					newPath = currentPath + '/' + folder.name;
				}
				html += '<div class="item-hover flex items-center gap-3 p-3 rounded-lg cursor-pointer border border-transparent hover:border-blue-200" onclick="navigateTo(\'' + newPath + '\')">';
				html += '<i class="fas fa-folder text-xl text-blue-500"></i>';
				html += '<span class="flex-1 text-sm text-blue-700 font-medium">' + folder.name + '</span>';
				html += '<i class="fas fa-chevron-right text-xs text-slate-400"></i>';
				html += '</div>';
			});
			
			// Show files
			data.files.forEach(file => {
				let fullPath;
				if (!currentPath || currentPath === '.' || currentPath === '') {
					fullPath = file.name;
				} else if (currentPath === '/') {
					fullPath = '/' + file.name;
				} else {
					fullPath = currentPath + '/' + file.name;
				}
				const isActive = currentFileName === fullPath;
				html += '<div class="item-hover flex items-center gap-2 p-2 rounded-lg border ' + (isActive ? 'border-blue-300 bg-blue-50' : 'border-transparent') + '">';
				html += '<i class="fas fa-database text-lg ' + (isActive ? 'text-blue-600' : 'text-slate-400') + '"></i>';
				html += '<button onclick="openFile(\'' + fullPath + '\')" class="flex-1 text-left text-sm ' + (isActive ? 'font-semibold text-blue-700' : 'text-slate-700') + '">' + file.name + '</button>';
				html += '<span class="text-xs text-slate-500 font-medium px-2 py-1 bg-slate-100 rounded">' + formatBytes(file.size) + '</span>';
				html += '<button onclick="deleteFile(\'' + fullPath + '\')" class="text-red-500 hover:text-red-700 hover:bg-red-50 p-2 rounded transition" title="Delete file"><i class="fas fa-trash-alt"></i></button>';
				html += '</div>';
			});
			container.innerHTML = html;
		}

	async function openFile(filename) {
		currentFileName = filename;
		await loadFile();
		await refreshFileList();
	}		async function deleteFile(filename) {
			if (!confirm('Are you sure you want to delete the file ' + filename + '?')) return;
			
			try {
				const response = await fetch('/api/files/' + encodeURIComponent(filename), {
					method: 'DELETE'
				});
				if (!response.ok) {
					throw new Error(await response.text());
				}
				if (currentFileName === filename) {
					currentFileName = null;
					document.getElementById('statsSection').classList.add('hidden');
					document.getElementById('recordsSection').classList.add('hidden');
				}
				showToast('File deleted successfully!', 'success');
				await refreshFileList();
			} catch (error) {
				showToast('Error deleting file: ' + error.message);
			}
		}

		function onEncryptionChange() {
			const encType = document.getElementById('encryptionType').value;
			const passwordField = document.getElementById('encryptionPasswordField');
			
			if (encType === 'none') {
				passwordField.classList.add('hidden');
				document.getElementById('encryptionPassword').value = '';
			} else {
				passwordField.classList.remove('hidden');
			}
		}

		async function refreshWithEncryption() {
			if (currentFileName) {
				await loadFile();
				showToast('Data refreshed with encryption options', 'success');
			}
		}

	async function loadFile() {
		if (!currentFileName) return;

		try {
			currentPage = 0; // Reset pagination
			document.getElementById('recordsContent').innerHTML = '<div class="text-center py-10"><div class="inline-block animate-spin rounded-full h-12 w-12 border-4 border-blue-200 border-t-blue-600"></div><p class="mt-4 text-sm text-slate-600"><i class="fas fa-cog fa-spin mr-2"></i>Loading records...</p></div>';
			document.getElementById('recordsSection').classList.remove('hidden');

			const encType = document.getElementById('encryptionType').value;
			const encPassword = document.getElementById('encryptionPassword').value;
			
			let url = '/api/parse?file=' + encodeURIComponent(currentFileName);
			if (encType !== 'none') {
				url += '&encryption=' + encType + '&password=' + encodeURIComponent(encPassword);
			}

			const response = await fetch(url);			if (!response.ok) {
				throw new Error(await response.text());
			}

			const data = await response.json();
			displayData(data);
		} catch (error) {
			showToast('Error: ' + error.message);
			document.getElementById('statsSection').classList.add('hidden');
			document.getElementById('recordsSection').classList.add('hidden');
		}
	}

	document.getElementById('searchInput').addEventListener('input', function(e) {
		const searchTerm = e.target.value.toLowerCase();
		const filtered = allRecords.filter(r => r.key.toLowerCase().includes(searchTerm));
	currentPage = 0; // Reset to first page on search
	displayRecords(filtered);
});		function displayData(data) {
			const statsGrid = document.getElementById('statsGrid');
			
			// Comprovar si la base de dades està corrupta
			if (data.stats.corrupted) {
				statsGrid.innerHTML = ` + "`" + `
					<div class="col-span-full flex flex-col items-center justify-center py-8 px-4">
						<div class="bg-gradient-to-br from-red-500 to-red-600 rounded-lg p-6 text-white shadow-xl max-w-2xl w-full">
							<div class="flex items-center gap-3 mb-4">
								<i class="fas fa-exclamation-triangle text-4xl"></i>
								<div class="flex-1">
									<div class="text-xl font-bold">Corrupted Database</div>
									<div class="text-sm opacity-90 mt-1">Cannot read data correctly</div>
								</div>
							</div>
							<div class="text-xs bg-red-700 bg-opacity-50 rounded p-3 mb-4 font-mono break-all">${data.stats.error}</div>
							<div class="text-sm mb-4"><i class="fas fa-hdd mr-2"></i>File size: ${formatBytes(data.stats.file_size)}</div>
							<div class="flex justify-center">
								<button onclick="showRecoverModal()" class="px-6 py-3 bg-white text-red-600 font-bold text-lg rounded-lg hover:bg-red-50 transition shadow-md">
									<i class="fas fa-medkit mr-2"></i>Recover Database
								</button>
							</div>
						</div>
					</div>
				` + "`" + `;
				document.getElementById('statsSection').classList.remove('hidden');
				document.getElementById('recordsSection').classList.add('hidden');
				return;
			}
			
			statsGrid.innerHTML = ` + "`" + `
				<div class="bg-gradient-to-br from-blue-500 to-blue-600 rounded-lg p-3 text-white shadow-md">
					<div class="flex items-center gap-2 mb-1.5">
						<i class="fas fa-list-ul"></i>
						<div class="text-xs font-medium opacity-90">Records</div>
					</div>
					<div class="text-2xl font-bold">${data.stats.active_records}</div>
				</div>
				<div class="bg-gradient-to-br from-red-500 to-red-600 rounded-lg p-3 text-white shadow-md">
					<div class="flex items-center gap-2 mb-1.5">
						<i class="fas fa-trash"></i>
						<div class="text-xs font-medium opacity-90">Deleted</div>
					</div>
					<div class="text-2xl font-bold">${data.stats.deleted_records}</div>
				</div>
				<div class="bg-gradient-to-br from-purple-500 to-purple-600 rounded-lg p-3 text-white shadow-md">
					<div class="flex items-center gap-2 mb-1.5">
						<i class="fas fa-hdd"></i>
						<div class="text-xs font-medium opacity-90">Size</div>
					</div>
					<div class="text-2xl font-bold">${formatBytes(data.stats.file_size)}</div>
				</div>
				<div class="bg-gradient-to-br from-green-500 to-green-600 rounded-lg p-3 text-white shadow-md">
					<div class="flex items-center gap-2 mb-1.5">
						<i class="fas fa-tachometer-alt"></i>
						<div class="text-xs font-medium opacity-90">Efficiency</div>
					</div>
					<div class="text-2xl font-bold">${data.stats.efficiency.toFixed(2)}%</div>
				</div>
				<div class="bg-gradient-to-br from-orange-500 to-orange-600 rounded-lg p-3 text-white shadow-md">
					<div class="flex items-center gap-2 mb-1.5">
						<i class="fas fa-exclamation-triangle"></i>
						<div class="text-xs font-medium opacity-90">Wasted</div>
					</div>
					<div class="text-2xl font-bold">${data.stats.wasted_percent.toFixed(2)}%</div>
			</div>
			<div class="bg-gradient-to-br from-indigo-500 to-indigo-600 rounded-lg p-3 text-white shadow-md">
				<div class="flex items-center gap-2 mb-1.5">
					<i class="fas fa-chart-line"></i>
					<div class="text-xs font-medium opacity-90">Avg Value</div>
				</div>
				<div class="text-2xl font-bold">${data.stats.average_data_size.toFixed(2)} B</div>
			</div>
		` + "`" + `;
		document.getElementById('statsSection').classList.remove('hidden');			allRecords = data.records;
			
			// Show warning if there are corrupted records
			const recordsSection = document.getElementById('recordsSection');
			if (data.stats.corrupted_count && data.stats.corrupted_count > 0) {
				const warningDiv = document.createElement('div');
				warningDiv.className = 'bg-yellow-50 border-l-4 border-yellow-400 p-4 mb-4';
				warningDiv.innerHTML = ` + "`" + `
					<div class="flex items-start">
						<i class="fas fa-exclamation-triangle text-yellow-400 mt-1 mr-3"></i>
						<div class="flex-1">
							<h3 class="text-sm font-bold text-yellow-800">
								Corrupted Records Detected
							</h3>
							<div class="mt-2 text-sm text-yellow-700">
								<p class="mb-2">${data.stats.corrupted_count} record(s) could not be read due to corruption:</p>
								<div class="font-mono text-xs bg-yellow-100 rounded p-2">
									${data.stats.corrupted_keys.join(', ')}
								</div>
								<p class="mt-2">
									These records have invalid data format and cannot be accessed. 
									Consider running database recovery to fix this issue.
								</p>
							</div>
						</div>
					</div>
				` + "`" + `;
				
				// Insert warning before records content
				const recordsContent = document.getElementById('recordsContent');
				recordsContent.parentNode.insertBefore(warningDiv, recordsContent);
			}
			
			displayRecords(allRecords);
		}

	function displayRecords(records) {
		const content = document.getElementById('recordsContent');
		
		if (records.length === 0) {
			content.innerHTML = '<div class="text-center py-8"><i class="fas fa-inbox text-4xl text-slate-300 mb-3"></i><p class="text-sm text-slate-500">No records found</p></div>';
			return;
		}

		// Calculate pagination
		const totalPages = Math.ceil(records.length / pageSize);
		const start = currentPage * pageSize;
		const end = Math.min(start + pageSize, records.length);
		const pageRecords = records.slice(start, end);

		let html = '<table class="min-w-full divide-y divide-slate-200"><thead class="bg-gradient-to-r from-slate-50 to-slate-100"><tr>';
		html += '<th class="px-4 py-2 text-left text-xs font-bold text-slate-700 uppercase tracking-wider"><i class="fas fa-key mr-2 text-blue-500"></i>Key</th>';
		html += '<th class="px-4 py-2 text-left text-xs font-bold text-slate-700 uppercase tracking-wider w-24"><i class="fas fa-tag mr-2 text-green-500"></i>Type</th>';
		html += '<th class="px-4 py-2 text-left text-xs font-bold text-slate-700 uppercase tracking-wider w-28"><i class="fas fa-weight-hanging mr-2 text-purple-500"></i>Size</th>';
		html += '<th class="px-4 py-2 text-left text-xs font-bold text-slate-700 uppercase tracking-wider"><i class="fas fa-file-alt mr-2 text-indigo-500"></i>Value</th>';
		html += '<th class="px-4 py-2 text-center text-xs font-bold text-slate-700 uppercase tracking-wider w-36"><i class="fas fa-cog mr-2 text-slate-500"></i>Actions</th>';
		html += '</tr></thead><tbody class="bg-white divide-y divide-slate-200">';

		pageRecords.forEach((record, idx) => {
			const actualIdx = start + idx; // Use actual index for editing
			const badge = record.is_text 
				? '<span class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold bg-green-100 text-green-800 rounded-full"><i class="fas fa-font"></i> Text</span>' 
				: '<span class="inline-flex items-center gap-1 px-2.5 py-1 text-xs font-semibold bg-orange-100 text-orange-800 rounded-full"><i class="fas fa-file-code"></i> Binary</span>';
			
			let displayValue = escapeHtml(record.value);
			if (displayValue.length > 100) {
				displayValue = displayValue.substring(0, 100) + '...';
			}

			html += '<tr class="hover:bg-blue-50 transition">';
			html += '<td class="px-4 py-2 text-sm font-semibold text-slate-900">' + escapeHtml(record.key) + '</td>';
			html += '<td class="px-4 py-2 text-sm">' + badge + '</td>';
			html += '<td class="px-4 py-2"><span class="text-xs text-slate-600 font-medium px-2 py-1 bg-slate-100 rounded">' + formatBytes(record.value_size) + '</span></td>';
			html += '<td class="px-4 py-2"><div class="text-sm font-mono text-slate-700 break-all">' + displayValue + '</div></td>';
			html += '<td class="px-4 py-2 text-center">';
			html += '<div class="flex items-center justify-center gap-2">';
			html += '<button onclick="editRecord(' + actualIdx + ')" class="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-lg transition"><i class="fas fa-edit"></i> Edit</button>';
			html += '<button onclick="showDeleteModal(\'' + escapeHtml(record.key) + '\')" class="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-red-700 bg-red-50 hover:bg-red-100 rounded-lg transition"><i class="fas fa-trash"></i> Delete</button>';
			html += '</div>';
			html += '</td>';
			html += '</tr>';
		});

		html += '</tbody></table>';
		
		// Add pagination controls
		if (totalPages > 1) {
			html += '<div class="flex items-center justify-between px-4 py-3 bg-slate-50 border-t border-slate-200">';
			html += '<div class="text-sm text-slate-600">';
			html += 'Showing ' + (start + 1) + ' to ' + end + ' of ' + records.length + ' records';
			html += '</div>';
			html += '<div class="flex gap-2">';
			
			// Previous button
			if (currentPage > 0) {
				html += '<button onclick="changePage(' + (currentPage - 1) + ')" class="px-3 py-1.5 text-sm font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-lg transition">';
				html += '<i class="fas fa-chevron-left mr-1"></i> Previous';
				html += '</button>';
			}
			
			// Page numbers
			const maxButtons = 5;
			let startPage = Math.max(0, currentPage - Math.floor(maxButtons / 2));
			let endPage = Math.min(totalPages, startPage + maxButtons);
			
			if (endPage - startPage < maxButtons) {
				startPage = Math.max(0, endPage - maxButtons);
			}
			
			for (let i = startPage; i < endPage; i++) {
				const activeClass = i === currentPage 
					? 'bg-blue-600 text-white' 
					: 'text-slate-700 bg-white hover:bg-slate-100';
				html += '<button onclick="changePage(' + i + ')" class="px-3 py-1.5 text-sm font-medium ' + activeClass + ' rounded-lg transition border border-slate-300">';
				html += (i + 1);
				html += '</button>';
			}
			
			// Next button
			if (currentPage < totalPages - 1) {
				html += '<button onclick="changePage(' + (currentPage + 1) + ')" class="px-3 py-1.5 text-sm font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-lg transition">';
				html += 'Next <i class="fas fa-chevron-right ml-1"></i>';
				html += '</button>';
			}
			
			html += '</div></div>';
		}
		
		content.innerHTML = html;
	}

	function changePage(page) {
		currentPage = page;
		displayRecords(allRecords);
	}		function showNewDbModal() {
			document.getElementById('newDbName').value = '';
			document.getElementById('newDbModal').classList.remove('hidden');
		}

		function closeNewDbModal() {
			document.getElementById('newDbModal').classList.add('hidden');
		}

		async function createNewDb() {
			const name = document.getElementById('newDbName').value.trim();
			if (!name) {
				showToast('Please enter a database name', 'error');
				return;
			}

			if (!name.endsWith('.skv')) {
				showToast('Database name must end with .skv', 'error');
				return;
			}

			try {
				let fullPath;
				if (!currentPath || currentPath === '.' || currentPath === '') {
					fullPath = name;
				} else if (currentPath === '/') {
					fullPath = '/' + name;
				} else {
					fullPath = currentPath + '/' + name;
				}
				const response = await fetch('/api/create', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ name: fullPath })
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				currentFileName = fullPath;
				
				closeNewDbModal();
				showToast('Database created successfully!', 'success');
				await refreshFileList();
				await loadFile();
			} catch (error) {
				showToast('Error creating database: ' + error.message);
			}
		}

		function showAddModal() {
			isAddMode = true;
			editingIndex = -1;
			document.getElementById('modalTitle').textContent = 'Add Record';
			document.getElementById('modalIcon').className = 'fas fa-plus-circle text-2xl';
			document.getElementById('editKey').value = '';
			document.getElementById('editKey').readOnly = false;
			document.getElementById('editValue').value = '';
			setEditMode('text');
			document.getElementById('editModal').classList.remove('hidden');
		}

		function editRecord(idx) {
			isAddMode = false;
			editingIndex = idx;
			const record = allRecords[idx];
			document.getElementById('modalTitle').textContent = 'Edit Record';
			document.getElementById('modalIcon').className = 'fas fa-edit text-2xl';
			document.getElementById('editKey').value = record.key;
			document.getElementById('editKey').readOnly = true;
			document.getElementById('editValue').value = record.value;
			setEditMode('text');
			document.getElementById('editModal').classList.remove('hidden');
		}

		function closeEditModal() {
			document.getElementById('editModal').classList.add('hidden');
			document.getElementById('hexError').classList.add('hidden');
		}

		function setEditMode(mode) {
			editMode = mode;
			const textBtn = document.getElementById('textModeBtn');
			const hexBtn = document.getElementById('hexModeBtn');
			const valueEl = document.getElementById('editValue');

			if (mode === 'text') {
				textBtn.className = 'px-3 py-1 text-xs font-medium rounded bg-gray-800 text-white';
				hexBtn.className = 'px-3 py-1 text-xs font-medium rounded bg-gray-200 text-gray-700';
				if (!isAddMode && editingIndex >= 0) {
					valueEl.value = allRecords[editingIndex].value;
				}
			} else {
				textBtn.className = 'px-3 py-1 text-xs font-medium rounded bg-gray-200 text-gray-700';
				hexBtn.className = 'px-3 py-1 text-xs font-medium rounded bg-gray-800 text-white';
				if (!isAddMode && editingIndex >= 0) {
					valueEl.value = allRecords[editingIndex].hex_value;
				}
			}
			document.getElementById('hexError').classList.add('hidden');
		}

		async function saveEdit() {
			const key = document.getElementById('editKey').value.trim();
			let value = document.getElementById('editValue').value;
			
			if (!key) {
				showToast('Key cannot be empty', 'error');
				return;
			}

			if (editMode === 'hex') {
				const hexPattern = /^[0-9a-fA-F]*$/;
				if (!hexPattern.test(value.replace(/\s/g, ''))) {
					document.getElementById('hexErrorText').textContent = 'Format hexadecimal invàlid';
					document.getElementById('hexError').classList.remove('hidden');
					return;
				}
			}

			const formData = new FormData();
			formData.append('filename', currentFileName);
			formData.append('key', key);
			formData.append('value', value);
			formData.append('is_hex', editMode === 'hex' ? 'true' : 'false');
			
			// Afegir paràmetres de xifrat
			const encType = document.getElementById('encryptionType').value;
			const encPassword = document.getElementById('encryptionPassword').value;
			if (encType !== 'none') {
				formData.append('encryption', encType);
				formData.append('password', encPassword);
			}

			const endpoint = isAddMode ? '/api/add' : '/api/update';

			try {
				const response = await fetch(endpoint, {
					method: 'POST',
					body: formData
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}
				
				closeEditModal();
				showToast(isAddMode ? 'Record added successfully!' : 'Record updated successfully!');
				await loadFile();
			} catch (error) {
				showToast('Error saving: ' + error.message);
			}
		}

		function showDeleteModal(key) {
			deletingKey = key;
			document.getElementById('deleteKeyName').textContent = key;
			document.getElementById('deleteModal').classList.remove('hidden');
		}

		function closeDeleteModal() {
			document.getElementById('deleteModal').classList.add('hidden');
			deletingKey = null;
		}

		async function confirmDelete() {
			if (!deletingKey) return;

			const formData = new FormData();
			formData.append('filename', currentFileName);
			formData.append('key', deletingKey);
			
			// Afegir paràmetres de xifrat
			const encType = document.getElementById('encryptionType').value;
			const encPassword = document.getElementById('encryptionPassword').value;
			if (encType !== 'none') {
				formData.append('encryption', encType);
				formData.append('password', encPassword);
			}

			try {
				const response = await fetch('/api/delete', {
					method: 'POST',
					body: formData
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}
				
				closeDeleteModal();
				showToast('Record deleted successfully!', 'success');
				await loadFile();
			} catch (error) {
				showToast('Error deleting: ' + error.message);
			}
		}

		// Bulk insert functions
		function showBulkInsertModal() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}
			document.getElementById('bulkInsertModal').classList.remove('hidden');
			resetBulkForm();
			updateBulkCount();
		}

		function closeBulkInsertModal() {
			document.getElementById('bulkInsertModal').classList.add('hidden');
		}

		function resetBulkForm() {
			const container = document.getElementById('bulkPairs');
			container.innerHTML = ` + "`" + `
				<div class="flex items-center gap-3 bulk-pair-row">
					<input type="text" placeholder="Key" class="flex-1 px-4 py-2.5 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-key">
					<span class="text-slate-400 font-bold">=</span>
					<input type="text" placeholder="Value" class="flex-1 px-4 py-2.5 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-value">
					<button onclick="removeBulkPair(this)" class="text-red-500 hover:text-red-700 hover:bg-red-50 p-2 rounded transition" title="Delete">
						<i class="fas fa-trash-alt"></i>
					</button>
				</div>
			` + "`" + `;
		}

		function addBulkPair() {
			const container = document.getElementById('bulkPairs');
			const newRow = document.createElement('div');
			newRow.className = 'flex items-center gap-3 bulk-pair-row';
			newRow.innerHTML = ` + "`" + `
				<input type="text" placeholder="Key" class="flex-1 px-4 py-2.5 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-key">
				<span class="text-slate-400 font-bold">=</span>
				<input type="text" placeholder="Value" class="flex-1 px-4 py-2.5 text-sm border-2 border-slate-200 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 outline-none font-mono bulk-value">
				<button onclick="removeBulkPair(this)" class="text-red-500 hover:text-red-700 hover:bg-red-50 p-2 rounded transition" title="Delete">
					<i class="fas fa-trash-alt"></i>
				</button>
			` + "`" + `;
			container.appendChild(newRow);
			updateBulkCount();
			// Focus on the new key input
			newRow.querySelector('.bulk-key').focus();
		}

		function removeBulkPair(button) {
			const rows = document.querySelectorAll('.bulk-pair-row');
			if (rows.length > 1) {
				button.closest('.bulk-pair-row').remove();
				updateBulkCount();
			}
		}

		function updateBulkCount() {
			const count = document.querySelectorAll('.bulk-pair-row').length;
			document.getElementById('bulkCount').textContent = count;
		}

		async function saveBulkInsert() {
			const rows = document.querySelectorAll('.bulk-pair-row');
			const pairs = [];
			
			for (const row of rows) {
				const key = row.querySelector('.bulk-key').value.trim();
				const value = row.querySelector('.bulk-value').value;
				
				if (key) { // Only add non-empty keys
					pairs.push({ key, value });
				}
			}

			if (pairs.length === 0) {
				showToast('Please add at least one key-value pair', 'error');
				return;
			}

		let successCount = 0;
		let errorCount = 0;

		for (const pair of pairs) {
			const formData = new FormData();
			formData.append('filename', currentFileName);
			formData.append('key', pair.key);
			formData.append('value', pair.value);
			formData.append('is_hex', 'false');
			
			// Afegir paràmetres de xifrat
			const encType = document.getElementById('encryptionType').value;
			const encPassword = document.getElementById('encryptionPassword').value;
			if (encType !== 'none') {
				formData.append('encryption', encType);
				formData.append('password', encPassword);
			}

			try {
				const response = await fetch('/api/add', {
					method: 'POST',
					body: formData
				});

				if (response.ok) {
					successCount++;
				} else {
					errorCount++;
				}
			} catch (error) {
				errorCount++;
			}
		}			closeBulkInsertModal();
			
			if (errorCount > 0) {
				showToast('Inserted ' + successCount + ' records. ' + errorCount + ' errors.', 'success');
			} else {
				showToast('All ' + successCount + ' inserted successfully!', 'success');
			}
			
			await loadFile();
		}

		// Backup modal functions
		function showBackupModal() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}
			const filename = currentFileName.split('/').pop().replace('.skv', '');
			document.getElementById('backupName').value = filename + '_backup.json';
			document.getElementById('backupModal').classList.remove('hidden');
		}

		function closeBackupModal() {
			document.getElementById('backupModal').classList.add('hidden');
		}

		async function createBackup() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}

			const backupName = document.getElementById('backupName').value.trim();
			if (!backupName) {
				showToast('Please enter a name for the backup', 'error');
				return;
			}

			if (!backupName.endsWith('.json')) {
				showToast('Filename must end with .json', 'error');
				return;
			}

			try {
				const response = await fetch('/api/backup', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ 
						filename: currentFileName,
						backup_name: backupName
					})
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				// Download the backup file
				const blob = await response.blob();
				const downloadUrl = window.URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = downloadUrl;
				a.download = backupName;
				document.body.appendChild(a);
				a.click();
				document.body.removeChild(a);
				window.URL.revokeObjectURL(downloadUrl);

				closeBackupModal();
				showToast('Backup created and downloaded successfully!', 'success');
			} catch (error) {
				showToast('Error creant backup: ' + error.message);
			}
		}

		// Restore modal functions
		function showRestoreModal() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}
			document.getElementById('restoreFile').value = '';
			document.getElementById('restoreModal').classList.remove('hidden');
		}

		function closeRestoreModal() {
			document.getElementById('restoreModal').classList.add('hidden');
		}

		async function restoreBackup() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}

			const fileInput = document.getElementById('restoreFile');
			if (!fileInput.files || fileInput.files.length === 0) {
				showToast('Please select a JSON file', 'error');
				return;
			}

			const file = fileInput.files[0];
			if (!file.name.endsWith('.json')) {
				showToast('Please select a valid JSON file', 'error');
				return;
			}

			if (!confirm('Do you want to restore the database from this backup? Existing keys with the same name will be overwritten.')) {
				return;
			}

			try {
				const formData = new FormData();
				formData.append('filename', currentFileName);
				formData.append('backup_file', file);

				const response = await fetch('/api/restore', {
					method: 'POST',
					body: formData
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				const result = await response.text();
				closeRestoreModal();
				showToast(result);
				await loadFile();
			} catch (error) {
				showToast('Error restaurant backup: ' + error.message);
			}
		}

		// Recover modal functions
		function showRecoverModal() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}
			// Obtenir el directori i nom del fitxer original
			const parts = currentFileName.split('/');
			const filename = parts[parts.length - 1].replace('.skv', '');
			const dir = parts.slice(0, -1).join('/');
			// Crear la ruta completa al mateix directori
			const recoveredPath = (dir ? dir + '/' : '') + filename + '_recuperada.skv';
			document.getElementById('recoverName').value = recoveredPath;
			document.getElementById('recoverModal').classList.remove('hidden');
		}

		function closeRecoverModal() {
			document.getElementById('recoverModal').classList.add('hidden');
		}

		async function recoverDatabase() {
			if (!currentFileName) {
				showToast('Please select a database first', 'error');
				return;
			}

			const recoverName = document.getElementById('recoverName').value.trim();
			if (!recoverName) {
				showToast('Please enter a name for the recovered database', 'error');
				return;
			}

			if (!recoverName.endsWith('.skv')) {
				showToast('Filename must end with .skv', 'error');
				return;
			}

			if (!confirm('Do you want to recover valid records from the current database? This will create a new database with recoverable records.')) {
				return;
			}

			try {
				const response = await fetch('/api/recover', {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ 
						corrupted_file: currentFileName,
						recovered_file: recoverName
					})
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				const result = await response.json();
				closeRecoverModal();
				
				// Mostrar resum detallat de la recuperació
				const summary = 'RESUM DE LA RECUPERACIÓ\n' +
					'═══════════════════════════\n\n' +
					'✓ Recovered records: ' + result.recovered_count + '\n' +
					'⊗ Bytes scanned: ' + result.total_scanned + '\n' +
					'✓ Arxiu creat: ' + recoverName + '\n\n' +
					'Recovered database has been created successfully.';
				
				showToast(summary, 'success');
				showToast('Recovery completed: ' + result.recovered_count + ' records recovered', 'success');
				await refreshFileList();
			} catch (error) {
				showToast('Error recuperant: ' + error.message);
			}
		}

		// Compact database function
		async function compactDatabase() {
			if (!currentFileName) {
				showToast('Please select a database first');
				return;
			}

			if (!confirm('Do you want to compact the database? This will remove wasted space from deleted records.')) {
				return;
			}

			try {
				const formData = new FormData();
				formData.append('filename', currentFileName);
				
				// Add encryption params if present
				const encType = document.getElementById('encryptionType')?.value;
				const encPass = document.getElementById('encryptionPassword')?.value;
				if (encType && encType !== 'none' && encPass) {
					formData.append('encryption', encType);
					formData.append('password', encPass);
				}

				const response = await fetch('/api/compact', {
					method: 'POST',
					body: formData
				});

				if (!response.ok) {
					throw new Error(await response.text());
				}

				showToast('Database compacted successfully!', 'success');
				await loadFile();
			} catch (error) {
				showToast('Error compacting: ' + error.message);
			}
		}

		function formatBytes(bytes) {
			if (bytes < 1024) return bytes + ' B';
			const k = 1024;
			const sizes = ['B', 'KB', 'MB', 'GB'];
			const i = Math.floor(Math.log(bytes) / Math.log(k));
			return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i];
		}

		function escapeHtml(text) {
			const div = document.createElement('div');
			div.textContent = text;
			return div.innerHTML;
		}
	</script>
</body>
</html>
`
	return c.HTML(http.StatusOK, htmlContent)
}

// Llistar fitxers disponibles
func listFilesHandler(c echo.Context) error {
	relPath := c.QueryParam("path")

	// Base directory
	baseDir := "."
	if relPath != "" {
		baseDir = relPath
	}

	files, err := os.ReadDir(baseDir)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error reading directory: "+err.Error())
	}

	type FileInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	}

	type DirInfo struct {
		Name string `json:"name"`
	}

	fileList := []FileInfo{}
	folderList := []DirInfo{}

	for _, entry := range files {
		// Skip hidden files for root directories only
		if baseDir == "." && strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.IsDir() {
			folderList = append(folderList, DirInfo{Name: entry.Name()})
		} else if strings.HasSuffix(entry.Name(), ".skv") {
			info, err := entry.Info()
			if err == nil {
				fileList = append(fileList, FileInfo{
					Name: entry.Name(),
					Size: info.Size(),
				})
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"files":   fileList,
		"folders": folderList,
	})
}

// Delete fitxer
func deleteFileHandler(c echo.Context) error {
	encodedPath := c.Param("filename")
	if encodedPath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	// Decodificar URL encoding
	filepath, err := url.QueryUnescape(encodedPath)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid filename encoding: "+err.Error())
	}

	if !strings.HasSuffix(filepath, ".skv") {
		return c.String(http.StatusBadRequest, "Only .skv files allowed")
	}

	if err := os.Remove(filepath); err != nil {
		return c.String(http.StatusInternalServerError, "Error deleting file: "+err.Error())
	}

	return c.String(http.StatusOK, "File deleted successfully")
}

// Upload handler - guardar fitxer
func uploadFileHandler(c echo.Context) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.String(http.StatusBadRequest, "Error rebent l'arxiu: "+err.Error())
	}

	if !strings.HasSuffix(file.Filename, ".skv") {
		return c.String(http.StatusBadRequest, "Only .skv files allowed")
	}

	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error opening l'arxiu: "+err.Error())
	}
	defer src.Close()

	// Crear directori si no existeix
	dir := filepath.Dir(file.Filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return c.String(http.StatusInternalServerError, "Cannot create directory: "+err.Error())
		}
	}

	filePath := file.Filename
	dstFile, err := os.Create(filePath)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error creating file: "+err.Error())
	}

	if _, err = io.Copy(dstFile, src); err != nil {
		dstFile.Close()
		return c.String(http.StatusInternalServerError, "Error copying l'arxiu: "+err.Error())
	}
	dstFile.Close()

	return c.String(http.StatusOK, "File uploaded successfully")
}

func parseFileHandler(c echo.Context) error {
	filepath := c.QueryParam("file")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	if !strings.HasSuffix(filepath, ".skv") {
		return c.String(http.StatusBadRequest, "Only .skv files allowed")
	}

	// Obtenir paràmetres de xifrat
	opts, err := getEncryptionOptions(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Definir tipus Record
	type Record struct {
		Key       string `json:"key"`
		Value     string `json:"value"`
		IsText    bool   `json:"is_text"`
		HexValue  string `json:"hex_value"`
		ValueSize int    `json:"value_size"`
	}

	var response map[string]interface{}

	// Utilitzar withDB per gestionar la connexió de forma segura
	err = withDB(filepath, opts, func(db *skv.SKV) error {
		// Obtenir estadístiques
		stats, err := db.Verify()
		if err != nil {
			// Si la base de dades està corrupta, intentem obtenir estadístiques parcials
			fileInfo, statErr := os.Stat(filepath)
			if statErr != nil {
				return fmt.Errorf("error verifying file: %v", err)
			}

			// Retornar estadístiques parcials amb informació de corrupció
			response = map[string]interface{}{
				"stats": map[string]interface{}{
					"total_records":     0,
					"active_records":    0,
					"deleted_records":   0,
					"file_size":         fileInfo.Size(),
					"efficiency":        0.0,
					"wasted_percent":    0.0,
					"average_key_size":  0.0,
					"average_data_size": 0.0,
					"corrupted":         true,
					"error":             err.Error(),
				},
				"records": []Record{},
			}
			return nil
		}

		// Utilitzar cursor per obtenir registres ordenats per clau
		records := []Record{}
		corruptedKeys := []string{}

		cursor := db.NewCursor(nil) // nil per obtenir tots els registres ordenats
		defer cursor.Close()

		for {
			key, value, err := cursor.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				// Skip corrupted records and track them
				if key != nil {
					corruptedKeys = append(corruptedKeys, string(key))
				}
				continue
			}
			records = append(records, Record{
				Key:       string(key),
				Value:     string(value),
				IsText:    isPrintable(value),
				HexValue:  hex.EncodeToString(value),
				ValueSize: len(value),
			})
		}

		// Add warning if there are corrupted records
		statsMap := map[string]interface{}{
			"total_records":     stats.TotalRecords,
			"active_records":    stats.ActiveRecords,
			"deleted_records":   stats.DeletedRecords,
			"file_size":         stats.FileSize,
			"efficiency":        stats.Efficiency,
			"wasted_percent":    stats.WastedPercent,
			"average_key_size":  stats.AverageKeySize,
			"average_data_size": stats.AverageDataSize,
		}

		if len(corruptedKeys) > 0 {
			statsMap["corrupted_keys"] = corruptedKeys
			statsMap["corrupted_count"] = len(corruptedKeys)
		}

		// Retornar dades en JSON
		response = map[string]interface{}{
			"stats":   statsMap,
			"records": records,
		}
		return nil
	})

	// Si hi ha error obrint la base de dades
	if err != nil {
		fileInfo, statErr := os.Stat(filepath)
		if statErr != nil {
			return c.String(http.StatusInternalServerError, "Error opening SKV file: "+err.Error())
		}

		// Retornar estadístiques parcials amb informació de corrupció
		response = map[string]interface{}{
			"stats": map[string]interface{}{
				"total_records":     0,
				"active_records":    0,
				"deleted_records":   0,
				"file_size":         fileInfo.Size(),
				"efficiency":        0.0,
				"wasted_percent":    0.0,
				"average_key_size":  0.0,
				"average_data_size": 0.0,
				"corrupted":         true,
				"error":             err.Error(),
			},
			"records": []Record{},
		}
	}

	return c.JSON(http.StatusOK, response)
}

// API per actualitzar un registre
func updateRecordHandler(c echo.Context) error {
	filepath := c.FormValue("filename")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	key := c.FormValue("key")
	value := c.FormValue("value")
	isHex := c.FormValue("is_hex") == "true"

	if key == "" {
		return c.String(http.StatusBadRequest, "Key is required")
	}

	opts, err := getEncryptionOptions(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	// Convert value if hex
	var valueBytes []byte
	if isHex {
		valueBytes, err = hex.DecodeString(strings.ReplaceAll(value, " ", ""))
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid hex format: "+err.Error())
		}
	} else {
		valueBytes = []byte(value)
	}

	err = withDB(filepath, opts, func(db *skv.SKV) error {
		return db.Update([]byte(key), valueBytes)
	})

	if err != nil {
		return c.String(http.StatusInternalServerError, "Error updating record: "+err.Error())
	}

	return c.String(http.StatusOK, "Record updated successfully")
}

// Crear nova base de dades
func createDbHandler(c echo.Context) error {
	var req struct {
		Name string `json:"name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request")
	}

	if req.Name == "" || !strings.HasSuffix(req.Name, ".skv") {
		return c.String(http.StatusBadRequest, "Invalid database name")
	}

	// Crear el path complet
	filePath := req.Name
	if !filepath.IsAbs(filePath) {
		filePath = filepath.Join(baseDir, filePath)
	}

	// Crear directori si no existeix
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return c.String(http.StatusInternalServerError, "Cannot create directory: "+err.Error())
	}

	// Comprovar si ja existeix
	if _, err := os.Stat(filePath); err == nil {
		return c.String(http.StatusBadRequest, "Database already exists")
	}

	// Crear base de dades buida - SKV crea el fitxer automàticament
	db, err := skv.OpenWithOptions(filePath, skv.DefaultOptions())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot create database: "+err.Error())
	}

	if err := db.Close(); err != nil {
		return c.String(http.StatusInternalServerError, "Cannot close database: "+err.Error())
	}

	return c.String(http.StatusOK, "Database created successfully")
}

// getEncryptionOptions crea les opcions de SKV amb els paràmetres de xifrat del context
func getEncryptionOptions(c echo.Context) (*skv.Options, error) {
	opts := skv.DefaultOptions()

	encryptionType := c.FormValue("encryption")
	if encryptionType == "" {
		encryptionType = c.QueryParam("encryption")
	}

	encryptionPassword := c.FormValue("password")
	if encryptionPassword == "" {
		encryptionPassword = c.QueryParam("password")
	}

	if encryptionType != "" && encryptionType != "none" {
		if encryptionPassword == "" {
			return nil, fmt.Errorf("password is required for encryption")
		}

		switch encryptionType {
		case "aes":
			opts.Encryption = skv.EncryptionAES
			opts.EncryptionPassword = encryptionPassword
		case "simplecipher":
			opts.Encryption = skv.EncryptionSimpleCipher
			opts.EncryptionPassword = encryptionPassword
		default:
			return nil, fmt.Errorf("invalid encryption type")
		}
	}

	return opts, nil
}

// Afegir nou registre
func addRecordHandler(c echo.Context) error {
	filepath := c.FormValue("filename")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	key := c.FormValue("key")
	valueStr := c.FormValue("value")
	isHex := c.FormValue("is_hex") == "true"

	if key == "" {
		return c.String(http.StatusBadRequest, "Key cannot be empty")
	}

	opts, err := getEncryptionOptions(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	var value []byte
	if isHex {
		value, err = hex.DecodeString(strings.ReplaceAll(valueStr, " ", ""))
		if err != nil {
			return c.String(http.StatusBadRequest, "Invalid hex value: "+err.Error())
		}
	} else {
		value = []byte(valueStr)
	}

	err = withDB(filepath, opts, func(db *skv.SKV) error {
		// Comprovar si la clau ja existeix
		_, err := db.Get([]byte(key))
		if err == nil {
			return fmt.Errorf("key already exists")
		}

		return db.Put([]byte(key), value)
	})

	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot add record: "+err.Error())
	}

	return c.String(http.StatusOK, "Record added successfully")
}

// Delete registre
func deleteRecordHandler(c echo.Context) error {
	filepath := c.FormValue("filename")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	key := c.FormValue("key")
	if key == "" {
		return c.String(http.StatusBadRequest, "Key cannot be empty")
	}

	opts, err := getEncryptionOptions(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = withDB(filepath, opts, func(db *skv.SKV) error {
		return db.Delete([]byte(key))
	})

	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot delete record: "+err.Error())
	}

	return c.String(http.StatusOK, "Record deleted successfully")
}

// Compact database handler
func compactDbHandler(c echo.Context) error {
	filepath := c.FormValue("filename")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	opts, err := getEncryptionOptions(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}

	err = withDB(filepath, opts, func(db *skv.SKV) error {
		return db.Compact()
	})

	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot compact database: "+err.Error())
	}

	return c.String(http.StatusOK, "Database compacted successfully")
}

// Recover handler
func recoverDbHandler(c echo.Context) error {
	var req struct {
		CorruptedFile string `json:"corrupted_file"`
		RecoveredFile string `json:"recovered_file"`
	}

	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request")
	}

	if req.CorruptedFile == "" || req.RecoveredFile == "" {
		return c.String(http.StatusBadRequest, "Both corrupted_file and recovered_file are required")
	}

	// Comprovar que el fitxer corrupte existeix
	if _, err := os.Stat(req.CorruptedFile); os.IsNotExist(err) {
		return c.String(http.StatusBadRequest, "Corrupted file does not exist")
	}

	// Comprovar que el fitxer recuperat no existeix
	if _, err := os.Stat(req.RecoveredFile); err == nil {
		return c.String(http.StatusBadRequest, "Recovered file already exists")
	}

	// Crear directori de destinació si no existeix
	dir := filepath.Dir(req.RecoveredFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return c.String(http.StatusInternalServerError, "Cannot create directory: "+err.Error())
	}

	// Llegir el fitxer corrupte
	corruptedData, err := os.ReadFile(req.CorruptedFile)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot read corrupted file: "+err.Error())
	}

	// Crear la base de dades recuperada
	recoveredDb, err := skv.OpenWithOptions(req.RecoveredFile, skv.DefaultOptions())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot create recovered database: "+err.Error())
	}
	defer recoveredDb.Close()

	// Escanejar byte a byte per trobar registres vàlids
	recoveredCount := 0
	position := int64(0)
	fileSize := int64(len(corruptedData))

	// Saltar header si existeix
	if fileSize >= 6 && string(corruptedData[0:3]) == "SKV" {
		position = 6
	}

	// Escanejar per registres vàlids
	for position < fileSize {
		typeByte := corruptedData[position]
		baseType := typeByte & 0x0F

		// Comprovar si és un type byte vàlid
		if baseType != 0x01 && baseType != 0x02 && baseType != 0x04 && baseType != 0x08 {
			position++
			continue
		}

		// Intentar parsejar el registre
		record, recordSize, err := tryParseRecord(corruptedData, position, fileSize)
		if err != nil {
			position++
			continue
		}

		// Si el registre està eliminat, saltar-lo
		if record.deleted {
			position += recordSize
			continue
		}

		// Guardar el registre recuperat
		err = recoveredDb.Put(record.key, record.data)
		if err != nil {
			// Si la clau existeix, intentar actualitzar
			err = recoveredDb.Update(record.key, record.data)
			if err != nil {
				// Saltar aquest registre si no es pot guardar
				position += recordSize
				continue
			}
		}

		recoveredCount++
		position += recordSize
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"recovered_count": recoveredCount,
		"total_scanned":   fileSize,
	})
}

// Estructura per emmagatzemar informació del registre recuperat
type recordInfo struct {
	key     []byte
	data    []byte
	deleted bool
}

// Funció per intentar parsejar un registre
func tryParseRecord(data []byte, position int64, fileSize int64) (*recordInfo, int64, error) {
	if position >= fileSize {
		return nil, 0, fmt.Errorf("position out of bounds")
	}

	pos := position
	typeByte := data[pos]
	deleted := (typeByte & 0x80) != 0
	baseType := typeByte & 0x0F
	pos++

	// Llegir keySize
	if pos >= fileSize {
		return nil, 0, fmt.Errorf("incomplete record")
	}
	keySize := int(data[pos])
	pos++

	// Llegir key
	if pos+int64(keySize) > fileSize {
		return nil, 0, fmt.Errorf("incomplete key")
	}
	key := make([]byte, keySize)
	copy(key, data[pos:pos+int64(keySize)])
	pos += int64(keySize)

	// Llegir dataSize segons el type
	var dataSize int64
	switch baseType {
	case 0x01: // 1 byte
		if pos >= fileSize {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = int64(data[pos])
		pos++
	case 0x02: // 2 bytes
		if pos+1 >= fileSize {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = int64(data[pos]) | int64(data[pos+1])<<8
		pos += 2
	case 0x04: // 4 bytes
		if pos+3 >= fileSize {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = int64(data[pos]) | int64(data[pos+1])<<8 | int64(data[pos+2])<<16 | int64(data[pos+3])<<24
		pos += 4
	case 0x08: // 8 bytes
		if pos+7 >= fileSize {
			return nil, 0, fmt.Errorf("incomplete data size")
		}
		dataSize = int64(data[pos]) | int64(data[pos+1])<<8 | int64(data[pos+2])<<16 | int64(data[pos+3])<<24 |
			int64(data[pos+4])<<32 | int64(data[pos+5])<<40 | int64(data[pos+6])<<48 | int64(data[pos+7])<<56
		pos += 8
	default:
		return nil, 0, fmt.Errorf("invalid type")
	}

	// Comprovar que dataSize és raonable
	if dataSize < 0 || pos+dataSize > fileSize {
		return nil, 0, fmt.Errorf("invalid data size")
	}

	// Llegir data
	recordData := make([]byte, dataSize)
	copy(recordData, data[pos:pos+dataSize])
	pos += dataSize

	// Llegir CRC (2 o 4 bytes segons dataSize)
	crcSize := int64(2)
	if dataSize > 255 {
		crcSize = 4
	}

	if pos+crcSize > fileSize {
		return nil, 0, fmt.Errorf("incomplete CRC")
	}

	// Per simplificar, no verifiquem el CRC aquí
	// En producció caldria verificar-lo
	pos += crcSize

	recordSize := pos - position

	return &recordInfo{
		key:     key,
		data:    recordData,
		deleted: deleted,
	}, recordSize, nil
}

// Backup handler
func backupHandler(c echo.Context) error {
	var req struct {
		Filename   string `json:"filename"`
		BackupName string `json:"backup_name"`
	}

	if err := c.Bind(&req); err != nil {
		return c.String(http.StatusBadRequest, "Invalid request")
	}

	if req.Filename == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	if req.BackupName == "" {
		req.BackupName = "backup.json"
	}

	db, err := skv.OpenWithOptions(req.Filename, skv.DefaultOptions())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot open database: "+err.Error())
	}
	defer db.Close()

	// Create temporary file for backup
	tmpFile := "/tmp/" + req.BackupName
	if err := db.Backup(tmpFile); err != nil {
		return c.String(http.StatusInternalServerError, "Cannot create backup: "+err.Error())
	}
	defer os.Remove(tmpFile)

	// Send file to client
	return c.File(tmpFile)
}

// Restore handler
func restoreHandler(c echo.Context) error {
	filepath := c.FormValue("filename")
	if filepath == "" {
		return c.String(http.StatusBadRequest, "Filename is required")
	}

	// Get uploaded file
	file, err := c.FormFile("backup_file")
	if err != nil {
		return c.String(http.StatusBadRequest, "Backup file is required")
	}

	// Save uploaded file temporarily
	src, err := file.Open()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error opening uploaded file: "+err.Error())
	}
	defer src.Close()

	tmpFile := "/tmp/restore_" + file.Filename
	dst, err := os.Create(tmpFile)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error creating temporary file: "+err.Error())
	}

	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmpFile)
		return c.String(http.StatusInternalServerError, "Error saving file: "+err.Error())
	}
	dst.Close()
	defer os.Remove(tmpFile)

	// Tancar connexió en cache abans del restore per forçar recàrrega
	dbPoolMutex.Lock()
	if conn, exists := dbPool[filepath]; exists {
		conn.db.Close()
		delete(dbPool, filepath)
	}
	dbPoolMutex.Unlock()

	// Open database and restore
	db, err := skv.OpenWithOptions(filepath, skv.DefaultOptions())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Cannot open database: "+err.Error())
	}
	defer db.Close()

	if err := db.Restore(tmpFile); err != nil {
		return c.String(http.StatusInternalServerError, "Cannot restore backup: "+err.Error())
	}

	count := db.Count()
	return c.String(http.StatusOK, fmt.Sprintf("Backup restaurat correctament! (%d claus)", count))
}

// Funcions auxiliars
func isPrintable(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	for _, b := range data {
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
		if b > 126 {
			return false
		}
	}
	return true
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Heartbeat handler
func heartbeatHandler(c echo.Context) error {
	heartbeatMutex.Lock()
	lastHeartbeat = time.Now()
	heartbeatMutex.Unlock()
	return c.NoContent(http.StatusOK)
}

func main() {
	// Crear una nova instància d'Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Rutes
	e.GET("/", indexHandler)
	e.GET("/api/files", listFilesHandler)
	e.DELETE("/api/files/:filename", deleteFileHandler)
	e.POST("/api/upload", uploadFileHandler)
	e.GET("/api/parse", parseFileHandler)
	e.POST("/api/create", createDbHandler)
	e.POST("/api/add", addRecordHandler)
	e.POST("/api/update", updateRecordHandler)
	e.POST("/api/delete", deleteRecordHandler)
	e.POST("/api/compact", compactDbHandler)
	e.POST("/api/recover", recoverDbHandler)
	e.POST("/api/backup", backupHandler)
	e.POST("/api/restore", restoreHandler)
	e.POST("/api/heartbeat", heartbeatHandler)

	// Initialize heartbeat
	heartbeatMutex.Lock()
	lastHeartbeat = time.Now()
	heartbeatMutex.Unlock()

	// Monitor heartbeat in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cleanup idle connections periodically
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupIdleConnections()
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				heartbeatMutex.RLock()
				elapsed := time.Since(lastHeartbeat)
				heartbeatMutex.RUnlock()

				if elapsed > 5*time.Second {
					log.Println("Navegador tancat. Tancant servidor...")
					cancel()
					go func() {
						time.Sleep(100 * time.Millisecond)
						if err := e.Shutdown(context.Background()); err != nil {
							log.Printf("Error tancant servidor: %v", err)
						}
					}()
					return
				}
			}
		}
	}()

	// Iniciar el navegador en una goroutine
	go func() {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("Obrint navegador a: %s\n", serverURL)
		if err := openBrowser(serverURL); err != nil {
			log.Printf("Error opening el navegador: %v", err)
		}
	}()

	// Iniciar el servidor HTTP amb Echo
	fmt.Printf("🚀 Servidor escoltant a %s\n", port)

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := e.Start(port); err != nil && err != http.ErrServerClosed {
			log.Fatal("Error del servidor: ", err)
		}
	}()

	select {
	case <-quit:
		log.Println("Rebut senyal d'interrupció...")
	case <-ctx.Done():
		log.Println("Tancant per inactivitat del navegador...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	// Tancar totes les connexions de BD abans de tancar el servidor
	log.Println("Tancant connexions de bases de dades...")
	closeAllConnections()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error durant el tancament: %v\n", err)
	}
	log.Println("Servidor tancat correctament")
}
