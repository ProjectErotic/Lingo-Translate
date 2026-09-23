import React, { useEffect, useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  Button,
  Input,
} from "@/ui";
import {
  DeployService,
  SettingsService,
} from "@bindings/lingo-translate/cmd/lingo-desktop";
import type { PublishResult } from "@bindings/lingo-translate/pkg/plugins/chanomhub";
import { toast } from "sonner";
import { Share2, Loader2, ExternalLink } from "lucide-react";

interface PublishDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  gameDir: string;
}

export const PublishDialog: React.FC<PublishDialogProps> = ({
  open,
  onOpenChange,
  gameDir,
}) => {
  const [slug, setSlug] = useState("");
  const [token, setToken] = useState("");
  const [language, setLanguage] = useState("Thai");
  const [uploading, setUploading] = useState(false);
  const [result, setResult] = useState<PublishResult | null>(null);

  useEffect(() => {
    if (open) {
      SettingsService.GetSettings().then((s) => {
        if (s?.chanomhub_token) {
          setToken(s.chanomhub_token);
        }
      });
    }
  }, [open]);

  const handlePublish = async () => {
    if (!slug || !token) {
      toast.error("Game slug and Chanomhub token are required");
      return;
    }

    try {
      setUploading(true);
      const res = await DeployService.Publish({
        game_dir: gameDir,
        slug: slug,
        token: token,
        language: language,
        api_base: "",
        storage_url: "",
      });
      setResult(res);
      toast.success("Translation mod published to Chanomhub!");
    } catch (err: any) {
      toast.error(`Publishing failed: ${err?.message || err}`);
    } finally {
      setUploading(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-base">
            <Share2 className="w-5 h-5 text-primary" />
            Publish Mod to Chanomhub
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2 text-sm">
          <p className="text-xs text-muted-foreground">
            Automatically packages <code className="text-primary font-mono">lingo_translations/</code> into a zip archive and uploads it directly to Chanomhub mod moderation.
          </p>

          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              Chanomhub Game Article Slug
            </label>
            <Input
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              placeholder="e.g. final-fantasy-vii-remake"
              disabled={uploading}
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              API Token (JWT)
            </label>
            <Input
              type="password"
              value={token}
              onChange={(e) => setToken(e.target.value)}
              placeholder="Enter Chanomhub user JWT token"
              disabled={uploading}
            />
          </div>

          <div>
            <label className="block text-xs font-semibold text-muted-foreground mb-1">
              Language Tag
            </label>
            <Input
              value={language}
              onChange={(e) => setLanguage(e.target.value)}
              placeholder="Thai"
              disabled={uploading}
            />
          </div>

          {result && (
            <div className="border border-emerald-500/30 bg-emerald-500/10 p-3 rounded-md text-xs space-y-1">
              <span className="font-semibold text-emerald-400 block">
                {result.message}
              </span>
              <div className="text-muted-foreground">
                Archive Size: {(result.file_size_bytes / 1024).toFixed(1)} KB
              </div>
              {result.download_url && (
                <a
                  href={result.download_url}
                  target="_blank"
                  rel="noreferrer"
                  className="inline-flex items-center gap-1 text-primary hover:underline pt-1"
                >
                  <ExternalLink className="w-3.5 h-3.5" />
                  View Uploaded File
                </a>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="secondary"
            onClick={() => onOpenChange(false)}
            disabled={uploading}
          >
            {result ? "Close" : "Cancel"}
          </Button>
          {!result && (
            <Button
              onClick={handlePublish}
              disabled={uploading || !slug || !token}
              className="gap-1.5"
            >
              {uploading ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  Uploading…
                </>
              ) : (
                <>
                  <Share2 className="w-4 h-4" />
                  Upload & Submit
                </>
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
};
