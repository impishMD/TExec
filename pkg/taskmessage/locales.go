package taskmessage

import "strings"

type words struct{ Run, Success, Error, Deployment, Timeout, Test string }

// These locale keys match the account/interface dictionaries in web/src/lang.
var translations = map[string]words{
	"en":    {"Run", "Success", "Error", "Deployment", "Connection timed out", "Test notification (sample data)"},
	"ru":    {"Запуск", "Успешно", "Ошибка", "Развёртывание", "Превышено время ожидания соединения", "Тестовое оповещение (пример данных)"},
	"cs":    {"Spuštění", "Úspěch", "Chyba", "Nasazení", "Vypršel časový limit připojení", "Testovací oznámení (ukázková data)"},
	"de":    {"Ausführung", "Erfolgreich", "Fehler", "Bereitstellung", "Zeitüberschreitung der Verbindung", "Testbenachrichtigung (Beispieldaten)"},
	"es":    {"Ejecución", "Éxito", "Error", "Despliegue", "Se agotó el tiempo de conexión", "Notificación de prueba (datos de ejemplo)"},
	"fr":    {"Exécution", "Réussite", "Erreur", "Déploiement", "Délai de connexion dépassé", "Notification de test (exemple de données)"},
	"it":    {"Esecuzione", "Successo", "Errore", "Distribuzione", "Timeout della connessione", "Notifica di prova (dati di esempio)"},
	"ja":    {"実行", "成功", "エラー", "デプロイ", "接続がタイムアウトしました", "テスト通知（サンプルデータ）"},
	"ko":    {"실행", "성공", "오류", "배포", "연결 시간이 초과되었습니다", "테스트 알림 (샘플 데이터)"},
	"nl":    {"Uitvoering", "Geslaagd", "Fout", "Uitrol", "Verbindingstime-out", "Testmelding (voorbeeldgegevens)"},
	"pl":    {"Uruchomienie", "Sukces", "Błąd", "Wdrożenie", "Przekroczono limit czasu połączenia", "Powiadomienie testowe (przykładowe dane)"},
	"pt":    {"Execução", "Sucesso", "Erro", "Implementação", "Tempo limite de ligação excedido", "Notificação de teste (dados de exemplo)"},
	"pt_br": {"Execução", "Sucesso", "Erro", "Implantação", "Tempo limite de conexão excedido", "Notificação de teste (dados de exemplo)"},
	"zh_cn": {"运行", "成功", "错误", "部署", "连接超时", "测试通知（示例数据）"},
	"zh_tw": {"執行", "成功", "錯誤", "部署", "連線逾時", "測試通知（範例資料）"},
}

func NormalizeLocale(locale string) string {
	locale = strings.ReplaceAll(strings.ToLower(locale), "-", "_")
	if _, ok := translations[locale]; ok {
		return locale
	}
	base := strings.Split(locale, "_")[0]
	if _, ok := translations[base]; ok {
		return base
	}
	return "en"
}

func SampleValues(locale, project, author, status string) map[string]string {
	w := translations[NormalizeLocale(locale)]
	values := map[string]string{
		"project.name": project, "template.name": w.Deployment, "task.id": "42",
		"task.started_by": author, "task.url": "https://texec.example.com/project/1/templates/1?t=42",
		"task.message": w.Deployment, "task.version": "1.2.3",
	}
	if status == "error" {
		values["task.error"] = "[ERROR]: " + w.Timeout
	}
	return values
}

func TestPrefix(locale string) string {
	return "TExec: " + translations[NormalizeLocale(locale)].Test + "\n\n"
}
