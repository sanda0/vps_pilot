import { useState, useEffect } from 'react';
import { Github, Loader2, CheckCircle2, ExternalLink } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { githubApi } from '@/lib/api';
import { useToast } from '@/hooks/use-toast';

export default function GitHubSettings() {
  const [token, setToken] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [isChecking, setIsChecking] = useState(true);
  const { toast } = useToast();

  useEffect(() => {
    checkGitHubStatus();
  }, []);

  const checkGitHubStatus = async () => {
    try {
      const response = await githubApi.getStatus();
      setIsConnected(response.connected);
    } catch (error) {
      console.error('Failed to check GitHub status:', error);
      setIsConnected(false);
    } finally {
      setIsChecking(false);
    }
  };

  const handleSaveToken = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!token.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a GitHub token',
        variant: 'destructive',
      });
      return;
    }

    setIsLoading(true);
    try {
      const response = await githubApi.saveToken(token);
      toast({
        title: 'Success',
        description: response.message || 'GitHub connected successfully',
      });
      setIsConnected(true);
      setToken('');
    } catch (error: any) {
      toast({
        title: 'Error',
        description: error.response?.data?.error || 'Failed to save GitHub token',
        variant: 'destructive',
      });
    } finally {
      setIsLoading(false);
    }
  };

  const handleDisconnect = async () => {
    setIsLoading(true);
    try {
      await githubApi.deleteToken();
      toast({
        title: 'Success',
        description: 'GitHub disconnected successfully',
      });
      setIsConnected(false);
    } catch (error: any) {
      toast({
        title: 'Error',
        description: error.response?.data?.error || 'Failed to disconnect GitHub',
        variant: 'destructive',
      });
    } finally {
      setIsLoading(false);
    }
  };

  if (isChecking) {
    return (
      <div className="flex items-center justify-center h-64">
        <Loader2 className="h-8 w-8 animate-spin" />
      </div>
    );
  }

  return (
    <div className="container max-w-2xl py-8">
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Github className="h-6 w-6" />
            <CardTitle>GitHub Integration</CardTitle>
          </div>
          <CardDescription>
            Connect your GitHub account to easily select repositories when creating projects
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          {isConnected ? (
            <Alert>
              <CheckCircle2 className="h-4 w-4" />
              <AlertDescription>
                GitHub is connected. You can now select repositories from your account when creating projects.
              </AlertDescription>
            </Alert>
          ) : (
            <Alert>
              <AlertDescription>
                Create a Personal Access Token to connect your GitHub account.
              </AlertDescription>
            </Alert>
          )}

          {!isConnected && (
            <div className="space-y-4">
              <div className="rounded-lg border p-4 space-y-3 text-sm">
                <h4 className="font-medium">How to create a GitHub Personal Access Token:</h4>
                <ol className="list-decimal list-inside space-y-2 text-muted-foreground">
                  <li>
                    Go to{' '}
                    <a
                      href="https://github.com/settings/tokens/new?scopes=repo&description=VPS%20Pilot"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-primary hover:underline inline-flex items-center gap-1"
                    >
                      GitHub Token Settings
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  </li>
                  <li>Set a note like "VPS Pilot"</li>
                  <li>Select the <code className="bg-muted px-1 py-0.5 rounded">repo</code> scope</li>
                  <li>Click "Generate token"</li>
                  <li>Copy the token and paste it below</li>
                </ol>
              </div>

              <form onSubmit={handleSaveToken} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="token">Personal Access Token</Label>
                  <Input
                    id="token"
                    type="password"
                    placeholder="ghp_xxxxxxxxxxxxxxxxxxxx"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                    disabled={isLoading}
                  />
                </div>
                <Button type="submit" disabled={isLoading} className="w-full">
                  {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                  <Github className="mr-2 h-4 w-4" />
                  Connect GitHub
                </Button>
              </form>
            </div>
          )}

          {isConnected && (
            <div className="flex justify-end">
              <Button
                variant="destructive"
                onClick={handleDisconnect}
                disabled={isLoading}
              >
                {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Disconnect GitHub
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
