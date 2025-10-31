import os

def flatten_directory(source_dir, output_file="flattened_output.txt", include_exts=None, exclude_exts='.git', max_file_size=500_000):
    """
    Flattens a directory by writing all file contents and paths into a single text file.

    Args:
        source_dir (str): Path to the directory to flatten.
        output_file (str): Output text file path.
        include_exts (list[str], optional): File extensions to include (e.g., ['.py', '.js']).
        exclude_exts (list[str], optional): File extensions to exclude.
        max_file_size (int): Skip files larger than this (in bytes).
    """
    with open(output_file, "w", encoding="utf-8") as out:
        out.write(f"Flattened view of: {os.path.abspath(source_dir)}\n")
        out.write("=" * 80 + "\n\n")

        for root, dirs, files in os.walk(source_dir):
            rel_path = os.path.relpath(root, source_dir)
            out.write(f"\n📁 Directory: {rel_path}\n")
            out.write("-" * 80 + "\n")

            for file in files:
                file_path = os.path.join(root, file)
                ext = os.path.splitext(file)[1].lower()

                # Extension filtering
                if include_exts and ext not in include_exts:
                    continue
                if exclude_exts and ext in exclude_exts:
                    continue

                # Skip large files
                if os.path.getsize(file_path) > max_file_size:
                    out.write(f"[Skipped: {file} — too large]\n")
                    continue

                try:
                    with open(file_path, "r", encoding="utf-8") as f:
                        content = f.read()
                    out.write(f"\n--- FILE: {file_path} ---\n")
                    out.write(content)
                    out.write("\n--- END OF FILE ---\n\n")
                except Exception as e:
                    out.write(f"[Error reading {file_path}: {e}]\n")

    print(f"\n✅ Flattened directory written to: {output_file}")

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="Flatten a code directory into one text file.")
    parser.add_argument("directory", help="Path to directory to flatten")
    parser.add_argument("-o", "--output", default="flattened_output.txt", help="Output text file name")
    parser.add_argument("--include", nargs="*", help="File extensions to include (e.g., .py .js .html)")
    parser.add_argument("--exclude", nargs="*", help="File extensions to exclude")
    args = parser.parse_args()

    flatten_directory(
        source_dir=args.directory,
        output_file=args.output,
        include_exts=args.include,
        exclude_exts=args.exclude
    )
