export interface Issue {
  number: number;
  title: string;
  body: string;
  repository: string;
  url: string;
}

export interface CodeChanges {
  files: Record<string, string>;
  new_files: string[];
  deleted_files: string[];
  summary: string;
}
