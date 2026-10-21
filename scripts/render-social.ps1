Add-Type -AssemblyName System.Drawing

# The social card: the app's own player shot on the app's own page colour, with
# the promise set in the app's ink. 1280x640 is what Twitter and Open Graph want.
$w, $h = 1280, 640
$bg = [System.Drawing.ColorTranslator]::FromHtml("#fcfbf9")
$ink = [System.Drawing.ColorTranslator]::FromHtml("#33332f")
$muted = [System.Drawing.ColorTranslator]::FromHtml("#72726a")
$accent = [System.Drawing.ColorTranslator]::FromHtml("#bd481b")

$bmp = New-Object System.Drawing.Bitmap($w, $h)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::ClearTypeGridFit
$g.Clear($bg)

# The screenshot, cropped to the player and rounded by drawing it inside a
# rounded path.
$shot = [System.Drawing.Image]::FromFile("Y:\Learn\Go\youtube-playlist\docs\screenshots\player.png")
$shotW = 720
$shotH = [int]($shot.Height * ($shotW / $shot.Width))
$destX = $w - $shotW - 56
$destY = [int](($h - $shotH) / 2)

$radius = 18
$path = New-Object System.Drawing.Drawing2D.GraphicsPath
$path.AddArc($destX, $destY, $radius * 2, $radius * 2, 180, 90)
$path.AddArc($destX + $shotW - $radius * 2, $destY, $radius * 2, $radius * 2, 270, 90)
$path.AddArc($destX + $shotW - $radius * 2, $destY + $shotH - $radius * 2, $radius * 2, $radius * 2, 0, 90)
$path.AddArc($destX, $destY + $shotH - $radius * 2, $radius * 2, $radius * 2, 90, 90)
$path.CloseFigure()

$g.SetClip($path)
$g.DrawImage($shot, $destX, $destY, $shotW, $shotH)
$g.ResetClip()

# A hairline around it, so it reads as a surface rather than a cut-out.
$edge = New-Object System.Drawing.Pen([System.Drawing.ColorTranslator]::FromHtml("#e8e5df"), 1)
$g.DrawPath($edge, $path)

# The mark.
$mark = [System.Drawing.Image]::FromFile("Y:\Learn\Go\youtube-playlist\logo\icon-192.png")
$g.DrawImage($mark, 56, 56, 56, 56)

# The words.
$titleFont = New-Object System.Drawing.Font("Segoe UI", 46, [System.Drawing.FontStyle]::Bold)
$bodyFont = New-Object System.Drawing.Font("Segoe UI", 19, [System.Drawing.FontStyle]::Regular)
$nameFont = New-Object System.Drawing.Font("Segoe UI", 22, [System.Drawing.FontStyle]::Bold)
$inkBrush = New-Object System.Drawing.SolidBrush($ink)
$mutedBrush = New-Object System.Drawing.SolidBrush($muted)

$g.DrawString("Playlist", $nameFont, $inkBrush, 128, 64)
$g.DrawString("Paste a link.", $titleFont, $inkBrush, 56, 200)
$g.DrawString("It plays.", $titleFont, $inkBrush, 56, 262)
$g.DrawString("A native desktop player for YouTube", $bodyFont, $mutedBrush, 56, 356)
$g.DrawString("playlists. No browser, no ads.", $bodyFont, $mutedBrush, 56, 386)
$g.DrawString("Windows  ·  macOS  ·  Linux", $bodyFont, $mutedBrush, 56, 452)

$g.Dispose()
$bmp.Save("Y:\Learn\Go\youtube-playlist\site\assets\social.png", [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
$shot.Dispose()
$mark.Dispose()
"social.png written"
