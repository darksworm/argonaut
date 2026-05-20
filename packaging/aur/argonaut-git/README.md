# argonaut-git AUR package

Tracks `main` of https://github.com/darksworm/argonaut. Rebuilds from HEAD on every install.

## Publishing to AUR (one-time)

```sh
# 1. Clone the empty AUR repo (after creating the package page on aur.archlinux.org)
git clone ssh://aur@aur.archlinux.org/argonaut-git.git
cd argonaut-git

# 2. Copy PKGBUILD in
cp /path/to/argonaut/packaging/aur/argonaut-git/PKGBUILD .

# 3. Generate .SRCINFO and commit
makepkg --printsrcinfo > .SRCINFO
git add PKGBUILD .SRCINFO
git commit -m "initial commit: argonaut-git"
git push
```

## Installing

```sh
yay -S argonaut-git
# or
paru -S argonaut-git
```

To pull the latest `main`, reinstall: `yay -S argonaut-git`.
