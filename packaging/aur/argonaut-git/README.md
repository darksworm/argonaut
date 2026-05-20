# argonaut-git AUR package

Tracks `main` of https://github.com/darksworm/argonaut. Rebuilds from HEAD on every install.

## Publishing to AUR (one-time)

> **Do not re-run these steps for routine updates.** They are only for the
> initial package setup. Subsequent `pkgver` bumps and pushes are performed
> automatically by `.github/workflows/aur-git-publish.yml` on every push to
> `main`. To ship a change, merge it to `main` — do not push to the AUR repo
> by hand.

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
