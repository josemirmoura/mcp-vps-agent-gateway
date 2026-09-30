package gateway

import (
	"fmt"
	"os"
	"strings"
)

func gatewayLang() string {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv("VPS_AGENT_LANG")))
	raw = strings.ReplaceAll(raw, "_", "-")
	switch {
	case raw == "pt" || strings.HasPrefix(raw, "pt-br"):
		return "pt-BR"
	case raw == "es" || strings.HasPrefix(raw, "es-"):
		return "es"
	case raw == "de" || strings.HasPrefix(raw, "de-"):
		return "de"
	case raw == "fr" || strings.HasPrefix(raw, "fr-"):
		return "fr"
	case raw == "ja" || strings.HasPrefix(raw, "ja-"):
		return "ja"
	case raw == "id" || strings.HasPrefix(raw, "id-"):
		return "id"
	default:
		return "en"
	}
}

var serverInstructionsByLang = map[string]string{
	"en": `You are connected to Portico MCP, which operates this VPS only within authority defined by the operator. When beginning VPS work in this conversation, before claiming that you can open or modify projects, inspect permissions.list_root_access and, when useful, permissions.discover_scope. Briefly explain that physical_scope_root is only the AI's maximum physical ceiling, not permission to open its contents. Discovery may reveal only immediate directory names below the ceiling. To enter a project, request the least authority needed: read for inspection, work for read/create/edit plus confined shell, or compose when policy-allowed Compose operations are required. When the client supports MCP elicitation, authorization is presented through the client's native human-confirmation UI; never claim approval until that round trip has succeeded. The operator may deny or revoke delegations. Do not call work "root access" and do not confuse folder delegation with Linux root privilege. Protected files, including .env and .env.*, remain locked even inside an authorized folder; request permissions.request_sensitive_access only when the task genuinely requires that exact file and prefer the smallest temporary access. Never ask for the VPS SSH/root password. These instructions guide client UX; the Broker remains the authoritative security boundary.`,
	"pt-BR": `Você está conectado ao Portico MCP, que opera esta VPS somente dentro da autoridade definida pelo operador. Ao iniciar trabalho sobre a VPS nesta conversa, antes de afirmar que pode abrir ou alterar projetos, consulte permissions.list_root_access e, quando útil, permissions.discover_scope. Explique de forma breve que physical_scope_root é apenas o teto físico máximo da IA, não uma autorização para abrir seu conteúdo. A descoberta pode mostrar somente nomes das pastas imediatamente abaixo do teto. Para entrar em um projeto, solicite a menor delegação adequada: read para leitura, work para ler/criar/editar e shell confinado, ou compose quando operações Compose já permitidas pela policy forem necessárias. Quando o cliente oferecer elicitation MCP, a autorização aparece na interface nativa de confirmação humana; nunca afirme que houve aprovação antes de a rodada ser concluída. O operador pode negar ou revogar delegações. Não chame work de acesso root e não confunda delegação de pasta com privilégios Linux de root. Arquivos protegidos, incluindo .env e .env.*, permanecem trancados mesmo dentro de uma pasta autorizada; só solicite permissions.request_sensitive_access quando a tarefa realmente exigir esse arquivo exato e prefira acesso temporário mínimo. Nunca peça senha SSH/root da VPS. Estas instruções orientam a experiência; o Broker continua sendo a fronteira autoritativa de segurança.`,
	"es": `Estás conectado a Portico MCP, que opera esta VPS únicamente dentro de la autoridad definida por el operador. Antes de afirmar que puedes abrir o modificar proyectos, consulta permissions.list_root_access y, cuando sea útil, permissions.discover_scope. Explica brevemente que physical_scope_root es solo el techo físico máximo de la IA, no un permiso para abrir su contenido. El descubrimiento puede mostrar únicamente los nombres de los directorios inmediatamente bajo ese techo. Para entrar en un proyecto, solicita la menor autoridad necesaria: read para inspección, work para leer/crear/editar y usar shell confinado, o compose cuando se necesiten operaciones Compose ya permitidas por la policy. Cuando el cliente admita MCP elicitation, la autorización se presenta mediante su interfaz nativa de confirmación humana; nunca afirmes que fue aprobada antes de completar esa ronda. El operador puede denegar o revocar delegaciones. No llames a work "acceso root" ni confundas delegación de carpetas con privilegios root de Linux. Los archivos protegidos, incluidos .env y .env.*, permanecen bloqueados incluso dentro de una carpeta autorizada; solicita permissions.request_sensitive_access solo cuando la tarea necesite realmente ese archivo exacto y prefiere el acceso temporal mínimo. Nunca pidas la contraseña SSH/root de la VPS. Estas instrucciones guían la experiencia; el Broker sigue siendo la frontera de seguridad autoritativa.`,
	"de": `Sie sind mit Portico MCP verbunden. Portico arbeitet auf dieser VPS ausschließlich innerhalb der vom Operator festgelegten Berechtigungen. Bevor Sie behaupten, Projekte öffnen oder ändern zu können, prüfen Sie permissions.list_root_access und bei Bedarf permissions.discover_scope. Erklären Sie kurz, dass physical_scope_root nur die maximale physische Obergrenze der KI ist und keine Berechtigung zum Öffnen ihrer Inhalte. Discovery darf nur unmittelbare Verzeichnisnamen unterhalb dieser Obergrenze zeigen. Fordern Sie für ein Projekt stets die kleinste nötige Berechtigung an: read zur Inspektion, work zum Lesen/Erstellen/Bearbeiten plus eingeschränkter Shell oder compose, wenn bereits von der Policy erlaubte Compose-Aktionen benötigt werden. Unterstützt der Client MCP Elicitation, wird die Autorisierung über dessen native menschliche Bestätigungsoberfläche angezeigt; behaupten Sie niemals eine Genehmigung, bevor dieser Ablauf abgeschlossen ist. Der Operator kann Delegationen ablehnen oder widerrufen. Nennen Sie work nicht "Root-Zugriff" und verwechseln Sie Ordnerdelegation nicht mit Linux-root-Rechten. Geschützte Dateien einschließlich .env und .env.* bleiben selbst in autorisierten Ordnern gesperrt; fordern Sie permissions.request_sensitive_access nur an, wenn die Aufgabe genau diese Datei benötigt, und bevorzugen Sie minimalen temporären Zugriff. Fragen Sie niemals nach dem SSH/root-Passwort der VPS. Diese Hinweise steuern die UX; der Broker bleibt die maßgebliche Sicherheitsgrenze.`,
	"fr": `Vous êtes connecté à Portico MCP, qui n'agit sur cette VPS que dans les limites de l'autorité définie par l'opérateur. Avant d'affirmer que vous pouvez ouvrir ou modifier des projets, consultez permissions.list_root_access et, si utile, permissions.discover_scope. Expliquez brièvement que physical_scope_root est seulement le plafond physique maximal de l'IA, et non une autorisation d'ouvrir son contenu. La découverte ne peut révéler que les noms des répertoires immédiatement sous ce plafond. Pour entrer dans un projet, demandez l'autorité minimale nécessaire : read pour l'inspection, work pour lire/créer/modifier avec shell confiné, ou compose lorsque des opérations Compose déjà permises par la policy sont nécessaires. Lorsque le client prend en charge MCP elicitation, l'autorisation apparaît dans son interface native de confirmation humaine ; n'affirmez jamais qu'elle a été accordée avant la fin de cet échange. L'opérateur peut refuser ou révoquer les délégations. N'appelez pas work « accès root » et ne confondez pas délégation de dossier et privilèges root Linux. Les fichiers protégés, notamment .env et .env.*, restent verrouillés même dans un dossier autorisé ; ne demandez permissions.request_sensitive_access que si la tâche exige réellement ce fichier exact et privilégiez l'accès temporaire minimal. Ne demandez jamais le mot de passe SSH/root de la VPS. Ces instructions guident l'expérience ; le Broker reste la frontière de sécurité faisant autorité.`,
	"ja": `Portico MCP に接続されています。Portico はオペレーターが定義した権限の範囲内でのみこの VPS を操作します。プロジェクトを開いたり変更したりできると述べる前に、permissions.list_root_access を確認し、必要に応じて permissions.discover_scope を使用してください。physical_scope_root は AI が到達できる最大の物理上限であり、その内容を開く権限そのものではないことを簡潔に説明してください。Discovery が表示できるのは上限直下のディレクトリ名だけです。プロジェクトに入る場合は必要最小限の権限を要求してください。確認だけなら read、読み取り/作成/編集と制限付き shell が必要なら work、policy で既に許可された Compose 操作が必要なら compose です。クライアントが MCP elicitation をサポートする場合、承認はクライアント標準の人間向け確認 UI に表示されます。その往復が完了する前に承認済みと述べてはいけません。オペレーターは委任を拒否または取り消せます。work を「root access」と呼んだり、フォルダ委任と Linux root 権限を混同したりしないでください。.env や .env.* を含む保護ファイルは、許可済みフォルダ内でもロックされたままです。permissions.request_sensitive_access はその正確なファイルが本当に必要な場合だけ要求し、最小の一時アクセスを優先してください。VPS の SSH/root パスワードを求めてはいけません。これらは UX を導く指示であり、最終的なセキュリティ境界は Broker です。`,
	"id": `Anda terhubung ke Portico MCP, yang mengoperasikan VPS ini hanya dalam otoritas yang ditentukan operator. Sebelum menyatakan bahwa Anda dapat membuka atau mengubah proyek, periksa permissions.list_root_access dan, bila berguna, permissions.discover_scope. Jelaskan secara singkat bahwa physical_scope_root hanyalah batas fisik maksimum AI, bukan izin untuk membuka isinya. Discovery hanya boleh menampilkan nama direktori yang langsung berada di bawah batas tersebut. Untuk masuk ke proyek, minta otoritas paling kecil yang diperlukan: read untuk inspeksi, work untuk baca/buat/edit plus shell terbatas, atau compose bila operasi Compose yang sudah diizinkan policy diperlukan. Bila klien mendukung MCP elicitation, otorisasi ditampilkan melalui UI konfirmasi manusia bawaan klien; jangan pernah menyatakan persetujuan telah diberikan sebelum alur itu selesai. Operator dapat menolak atau mencabut delegasi. Jangan menyebut work sebagai "root access" dan jangan menyamakan delegasi folder dengan hak root Linux. File terlindungi, termasuk .env dan .env.*, tetap terkunci bahkan di dalam folder yang diotorisasi; minta permissions.request_sensitive_access hanya bila tugas benar-benar memerlukan file persis tersebut dan utamakan akses sementara sekecil mungkin. Jangan pernah meminta kata sandi SSH/root VPS. Instruksi ini memandu UX; Broker tetap menjadi batas keamanan yang berwenang.`,
}

type approvalLocale struct {
	rootTitle        string
	sensitiveTitle   string
	folderLabel      string
	fileLabel        string
	accessLabel      string
	durationLabel    string
	ceilingLabel     string
	rootBody         string
	sensitiveBody    string
	ceilingWarning   string
	revokeNotice     string
	readOnly         string
	work             string
	compose          string
	permanent        string
	secondsSuffix    string
}

var approvalLocales = map[string]approvalLocale{
	"en": {
		rootTitle: "Authorize Portico MCP access?", sensitiveTitle: "Authorize temporary access to a protected Portico MCP file?",
		folderLabel: "Folder", fileLabel: "File", accessLabel: "Access", durationLabel: "Duration", ceilingLabel: "Physical ceiling",
		rootBody: "The physical ceiling is the AI's maximum boundary; it grants no access by itself. This authorization enables only the profile above inside the requested folder. Protected files such as .env remain locked.",
		sensitiveBody: "Protected files such as .env stay locked even when the project folder is authorized. This exception applies only to the file above and expires automatically. Accept only if you intentionally want to expose this secret for the current task.",
		ceilingWarning: "WARNING: you are authorizing the physical ceiling itself. The requested profile will apply to every current and future folder below that ceiling while the authorization remains active.",
		revokeNotice: "You can revoke this authorization later.", readOnly: "Read only",
		work: "Work (read, create, edit, delete, and use confined shell)",
		compose: "Work + Compose actions already allowed by policy", permanent: "Permanent, until revoked", secondsSuffix: "sec",
	},
	"pt-BR": {
		rootTitle: "Autorizar acesso do Portico MCP?", sensitiveTitle: "Autorizar acesso temporário a um arquivo protegido do Portico MCP?",
		folderLabel: "Pasta", fileLabel: "Arquivo", accessLabel: "Acesso", durationLabel: "Duração", ceilingLabel: "Teto físico",
		rootBody: "O teto físico é o limite máximo da IA; ele não concede acesso por si só. Esta autorização libera somente o perfil acima dentro da pasta solicitada. Arquivos protegidos, como .env, continuam bloqueados.",
		sensitiveBody: "Arquivos protegidos, como .env, continuam bloqueados mesmo quando a pasta do projeto está autorizada. Esta exceção vale somente para o arquivo acima e expira automaticamente. Aceite apenas se você realmente quiser liberar esse segredo para a tarefa atual.",
		ceilingWarning: "ATENÇÃO: você está autorizando o próprio teto físico. O perfil solicitado passará a valer para todas as pastas atuais e futuras abaixo desse teto enquanto a autorização estiver ativa.",
		revokeNotice: "Você poderá revogar esta autorização depois.", readOnly: "Somente leitura",
		work: "Trabalho (ler, criar, editar, excluir e usar shell confinado)",
		compose: "Trabalho + Compose já permitido pela política", permanent: "Permanente, até revogação", secondsSuffix: "s",
	},
	"es": {
		rootTitle: "¿Autorizar acceso de Portico MCP?", sensitiveTitle: "¿Autorizar acceso temporal a un archivo protegido de Portico MCP?",
		folderLabel: "Carpeta", fileLabel: "Archivo", accessLabel: "Acceso", durationLabel: "Duración", ceilingLabel: "Techo físico",
		rootBody: "El techo físico es el límite máximo de la IA; por sí solo no concede acceso. Esta autorización habilita únicamente el perfil indicado dentro de la carpeta solicitada. Los archivos protegidos como .env siguen bloqueados.",
		sensitiveBody: "Los archivos protegidos como .env siguen bloqueados incluso cuando la carpeta del proyecto está autorizada. Esta excepción se aplica solo al archivo indicado y caduca automáticamente. Acepta únicamente si deseas exponer este secreto para la tarea actual.",
		ceilingWarning: "ADVERTENCIA: estás autorizando el propio techo físico. El perfil solicitado se aplicará a todas las carpetas actuales y futuras bajo ese techo mientras la autorización siga activa.",
		revokeNotice: "Puedes revocar esta autorización después.", readOnly: "Solo lectura",
		work: "Trabajo (leer, crear, editar, eliminar y usar shell confinado)",
		compose: "Trabajo + acciones Compose ya permitidas por la policy", permanent: "Permanente, hasta revocación", secondsSuffix: "s",
	},
	"de": {
		rootTitle: "Portico-MCP-Zugriff autorisieren?", sensitiveTitle: "Temporären Zugriff auf eine geschützte Portico-MCP-Datei autorisieren?",
		folderLabel: "Ordner", fileLabel: "Datei", accessLabel: "Zugriff", durationLabel: "Dauer", ceilingLabel: "Physische Obergrenze",
		rootBody: "Die physische Obergrenze ist die maximale Grenze der KI und gewährt allein keinen Zugriff. Diese Autorisierung aktiviert nur das oben genannte Profil im angeforderten Ordner. Geschützte Dateien wie .env bleiben gesperrt.",
		sensitiveBody: "Geschützte Dateien wie .env bleiben auch in einem autorisierten Projektordner gesperrt. Diese Ausnahme gilt nur für die oben genannte Datei und läuft automatisch ab. Akzeptieren Sie nur, wenn dieses Geheimnis für die aktuelle Aufgabe wirklich freigegeben werden soll.",
		ceilingWarning: "WARNUNG: Sie autorisieren die physische Obergrenze selbst. Das angeforderte Profil gilt dann für alle aktuellen und zukünftigen Ordner darunter, solange die Autorisierung aktiv ist.",
		revokeNotice: "Sie können diese Autorisierung später widerrufen.", readOnly: "Nur Lesen",
		work: "Arbeit (lesen, erstellen, bearbeiten, löschen und eingeschränkte Shell verwenden)",
		compose: "Arbeit + bereits von der Policy erlaubte Compose-Aktionen", permanent: "Permanent, bis zum Widerruf", secondsSuffix: "s",
	},
	"fr": {
		rootTitle: "Autoriser l'accès de Portico MCP ?", sensitiveTitle: "Autoriser temporairement l'accès à un fichier protégé de Portico MCP ?",
		folderLabel: "Dossier", fileLabel: "Fichier", accessLabel: "Accès", durationLabel: "Durée", ceilingLabel: "Plafond physique",
		rootBody: "Le plafond physique est la limite maximale de l'IA ; il n'accorde aucun accès à lui seul. Cette autorisation active uniquement le profil indiqué dans le dossier demandé. Les fichiers protégés tels que .env restent verrouillés.",
		sensitiveBody: "Les fichiers protégés tels que .env restent verrouillés même lorsque le dossier du projet est autorisé. Cette exception ne concerne que le fichier indiqué et expire automatiquement. Acceptez uniquement si vous souhaitez réellement exposer ce secret pour la tâche actuelle.",
		ceilingWarning: "AVERTISSEMENT : vous autorisez le plafond physique lui-même. Le profil demandé s'appliquera à tous les dossiers actuels et futurs sous ce plafond tant que l'autorisation restera active.",
		revokeNotice: "Vous pourrez révoquer cette autorisation plus tard.", readOnly: "Lecture seule",
		work: "Travail (lire, créer, modifier, supprimer et utiliser un shell confiné)",
		compose: "Travail + actions Compose déjà permises par la policy", permanent: "Permanent, jusqu'à révocation", secondsSuffix: "s",
	},
	"ja": {
		rootTitle: "Portico MCP のアクセスを許可しますか？", sensitiveTitle: "Portico MCP の保護ファイルへの一時アクセスを許可しますか？",
		folderLabel: "フォルダ", fileLabel: "ファイル", accessLabel: "アクセス", durationLabel: "期間", ceilingLabel: "物理上限",
		rootBody: "物理上限は AI が到達できる最大境界であり、それ自体はアクセス権を与えません。この承認は、指定されたフォルダ内で上記のプロファイルだけを有効にします。.env などの保護ファイルは引き続きロックされます。",
		sensitiveBody: ".env などの保護ファイルは、プロジェクトフォルダが許可されていてもロックされたままです。この例外は上記のファイルだけに適用され、自動的に期限切れになります。現在の作業のためにこのシークレットを意図的に公開する場合のみ承認してください。",
		ceilingWarning: "警告: 物理上限そのものを許可しようとしています。承認が有効な間、要求されたプロファイルはその上限以下の現在および将来のすべてのフォルダに適用されます。",
		revokeNotice: "この承認は後で取り消せます。", readOnly: "読み取りのみ",
		work: "作業（読み取り、作成、編集、削除、制限付き shell）",
		compose: "作業 + policy で既に許可された Compose 操作", permanent: "取り消すまで恒久", secondsSuffix: "秒",
	},
	"id": {
		rootTitle: "Izinkan akses Portico MCP?", sensitiveTitle: "Izinkan akses sementara ke file Portico MCP yang terlindungi?",
		folderLabel: "Folder", fileLabel: "File", accessLabel: "Akses", durationLabel: "Durasi", ceilingLabel: "Batas fisik",
		rootBody: "Batas fisik adalah batas maksimum AI; batas itu sendiri tidak memberikan akses. Otorisasi ini hanya mengaktifkan profil di atas di dalam folder yang diminta. File terlindungi seperti .env tetap terkunci.",
		sensitiveBody: "File terlindungi seperti .env tetap terkunci meskipun folder proyek sudah diotorisasi. Pengecualian ini hanya berlaku untuk file di atas dan kedaluwarsa otomatis. Terima hanya jika Anda memang ingin mengekspos secret ini untuk tugas saat ini.",
		ceilingWarning: "PERINGATAN: Anda sedang mengotorisasi batas fisik itu sendiri. Profil yang diminta akan berlaku untuk semua folder saat ini dan yang dibuat di masa depan di bawah batas tersebut selama otorisasi aktif.",
		revokeNotice: "Anda dapat mencabut otorisasi ini nanti.", readOnly: "Hanya baca",
		work: "Kerja (baca, buat, edit, hapus, dan gunakan shell terbatas)",
		compose: "Kerja + aksi Compose yang sudah diizinkan policy", permanent: "Permanen, sampai dicabut", secondsSuffix: "dtk",
	},
}

func approvalLocaleForCurrentLang() approvalLocale {
	if l, ok := approvalLocales[gatewayLang()]; ok {
		return l
	}
	return approvalLocales["en"]
}

func localizedApprovalMessage(kind string, values map[string]any) string {
	l := approvalLocaleForCurrentLang()
	access := stringValue(values["access"])
	duration := localizedDurationValue(values["delegation_ttl_seconds"])
	ceiling := stringValue(values["physical_ceiling"])

	if kind == "sensitive" {
		target := stringValue(values["path"])
		return fmt.Sprintf(
			"%s\n\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n\n%s",
			l.sensitiveTitle, l.fileLabel, target, l.accessLabel, localizedAccessLabel(access),
			l.durationLabel, duration, l.ceilingLabel, ceiling, l.sensitiveBody,
		)
	}

	target := stringValue(values["root"])
	message := fmt.Sprintf(
		"%s\n\n%s: %s\n%s: %s\n%s: %s\n%s: %s\n\n%s",
		l.rootTitle, l.folderLabel, target, l.accessLabel, localizedAccessLabel(access),
		l.durationLabel, duration, l.ceilingLabel, ceiling, l.rootBody,
	)
	if ceilingWide, _ := values["ceiling_wide"].(bool); ceilingWide {
		message += "\n\n" + l.ceilingWarning
	}
	return message + "\n\n" + l.revokeNotice
}

func localizedAccessLabel(access string) string {
	l := approvalLocaleForCurrentLang()
	switch access {
	case "read":
		return l.readOnly
	case "work":
		return l.work
	case "compose":
		return l.compose
	default:
		return access
	}
}

func localizedDurationValue(value any) string {
	l := approvalLocaleForCurrentLang()
	seconds := int64Value(value)
	if seconds <= 0 {
		return l.permanent
	}
	if seconds%3600 == 0 {
		return fmt.Sprintf("%d h", seconds/3600)
	}
	if seconds%60 == 0 {
		return fmt.Sprintf("%d min", seconds/60)
	}
	return fmt.Sprintf("%d %s", seconds, l.secondsSuffix)
}
