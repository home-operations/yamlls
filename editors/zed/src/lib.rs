use std::fs;

use zed_extension_api::{
    self as zed, settings::LspSettings, Architecture, DownloadedFileType, GithubReleaseOptions,
    LanguageServerId, Os, Result,
};

const REPO: &str = "home-operations/yamlls";

struct YamllsExtension {
    /// Cached path to the downloaded binary, so we don't re-check the release
    /// on every `language_server_command` call within a session.
    cached_binary_path: Option<String>,
}

impl YamllsExtension {
    fn binary_path(
        &mut self,
        id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String> {
        // 1. Explicit override: settings.json "binary": { "path": "..." }.
        if let Ok(lsp) = LspSettings::for_worktree("yamlls", worktree) {
            if let Some(path) = lsp.binary.and_then(|b| b.path) {
                return Ok(path);
            }
        }

        // 2. Reuse the binary downloaded earlier this session if it still exists.
        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).is_ok_and(|m| m.is_file()) {
                return Ok(path.clone());
            }
        }

        // 3. Resolve and download the matching release asset.
        zed::set_language_server_installation_status(
            id,
            &zed::LanguageServerInstallationStatus::CheckingForUpdate,
        );
        let release = zed::latest_github_release(
            REPO,
            GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        )?;

        let (os, arch) = zed::current_platform();
        let asset = asset_name(&release.version, os, arch)?;
        let asset_url = release
            .assets
            .iter()
            .find(|a| a.name == asset)
            .map(|a| a.download_url.clone())
            .ok_or_else(|| format!("no release asset named {asset} in {}", release.version))?;

        // GoReleaser archives carry the bare binary at the archive root.
        let version_dir = format!("yamlls-{}", release.version);
        let binary = format!("{version_dir}/{}", binary_name(os));

        if !fs::metadata(&binary).is_ok_and(|m| m.is_file()) {
            zed::set_language_server_installation_status(
                id,
                &zed::LanguageServerInstallationStatus::Downloading,
            );
            zed::download_file(&asset_url, &version_dir, archive_kind(os))?;
            zed::make_file_executable(&binary)?;

            // Keep only the current version's directory around.
            if let Ok(entries) = fs::read_dir(".") {
                for entry in entries.flatten() {
                    let name = entry.file_name();
                    let name = name.to_string_lossy();
                    if name.starts_with("yamlls-") && name != version_dir.as_str() {
                        fs::remove_dir_all(entry.path()).ok();
                    }
                }
            }
        }

        self.cached_binary_path = Some(binary.clone());
        Ok(binary)
    }
}

impl zed::Extension for YamllsExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        Ok(zed::Command {
            command: self.binary_path(id, worktree)?,
            args: vec![],
            env: vec![],
        })
    }

    fn language_server_initialization_options(
        &mut self,
        _id: &LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<Option<zed::serde_json::Value>> {
        // Forward the user's "initialization_options" block verbatim; the server
        // accepts schemas / catalog / catalogUrl / kubernetes / renderers.
        Ok(LspSettings::for_worktree("yamlls", worktree)
            .ok()
            .and_then(|s| s.initialization_options))
    }
}

fn binary_name(os: Os) -> &'static str {
    match os {
        Os::Windows => "yamlls.exe",
        _ => "yamlls",
    }
}

fn archive_kind(os: Os) -> DownloadedFileType {
    match os {
        Os::Windows => DownloadedFileType::Zip,
        _ => DownloadedFileType::GzipTar,
    }
}

/// Mirrors the GoReleaser asset names: yamlls_{version}_{os}_{arch}.{ext}
fn asset_name(version: &str, os: Os, arch: Architecture) -> Result<String> {
    let os_part = match os {
        Os::Mac => "darwin",
        Os::Linux => "linux",
        Os::Windows => "windows",
    };
    let arch_part = match arch {
        Architecture::Aarch64 => "arm64",
        Architecture::X8664 => "amd64",
        Architecture::X86 => return Err("yamlls has no 32-bit (x86) build".into()),
    };
    if os == Os::Windows && arch == Architecture::Aarch64 {
        return Err("yamlls has no windows/arm64 build".into());
    }
    let ext = if os == Os::Windows { "zip" } else { "tar.gz" };
    Ok(format!("yamlls_{version}_{os_part}_{arch_part}.{ext}"))
}

zed::register_extension!(YamllsExtension);
