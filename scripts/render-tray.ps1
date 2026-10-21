$ErrorActionPreference = "Stop"

# Render the tray icon in its two states.
#
# Playing and paused are the same mark; the paused one is dimmed to 45% alpha,
# which is how a tray shows that something is not running. Two files rather
# than one animated icon: a tray icon cannot animate on Windows or Linux, and
# swapping between two is what every player does.

$x0 = 39.0
$x1 = 234.0
$y0 = 59.0
$y1 = 197.0
$w = $x1 - $x0
$h = $y1 - $y0

$size = 256.0
$margin = 16.0
$target = $size - 2 * $margin
$s = [Math]::Min($target / $w, $target / $h)
$cx = ($x0 + $x1) / 2
$cy = ($y0 + $y1) / 2
$tx = $size / 2 - $cx * $s
$ty = $size / 2 - $cy * $s

$chrome = "C:\Program Files\Google\Chrome\Application\chrome.exe"
Add-Type -AssemblyName System.Drawing

function Render-State {
    param([string]$Name, [double]$Alpha)

    $svg = @"
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <rect width="256" height="256" fill="#ffffff"/>
  <g transform="translate($tx $ty) scale($s)">
    <path d="M 52 72 H 122 A 28 28 0 0 1 122 128 H 94 A 28 28 0 0 0 94 184 H 150"
          stroke="#000000" stroke-width="26" stroke-linecap="round" stroke-linejoin="round" fill="none"/>
    <path d="M 168 84 Q 168 74 178 74 L 184 74 Q 190 74 196 79 L 234 122 Q 240 128 234 134 L 196 177 Q 190 182 184 182 L 178 182 Q 168 182 168 172 Z" fill="#000000"/>
  </g>
</svg>
"@
    $svgPath = "Y:\Learn\Go\youtube-playlist\src\assets\tray-$Name.svg"
    $svg | Set-Content $svgPath -NoNewline

    $big = "Y:\Learn\Go\youtube-playlist\src\assets\tray-big.png"
    & $chrome --headless --disable-gpu --screenshot=$big --window-size=256,256 --hide-scrollbars "file:///$($svgPath -replace '\\','/')" 2>$null
    Start-Sleep -Seconds 2

    $src = [System.Drawing.Image]::FromFile($big)
    $small = New-Object System.Drawing.Bitmap 32, 32
    $g = [System.Drawing.Graphics]::FromImage($small)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.DrawImage($src, 0, 0, 32, 32)
    $g.Dispose()
    $src.Dispose()

    for ($y = 0; $y -lt 32; $y++) {
        for ($x = 0; $x -lt 32; $x++) {
            $c = $small.GetPixel($x, $y)
            $a = (255 - [int](($c.R + $c.G + $c.B) / 3)) * $Alpha
            $small.SetPixel($x, $y, [System.Drawing.Color]::FromArgb([int]$a, 255, 255, 255))
        }
    }
    $out = "Y:\Learn\Go\youtube-playlist\src\assets\tray-$Name.png"
    $small.Save($out, [System.Drawing.Imaging.ImageFormat]::Png)
    $small.Dispose()
    Remove-Item $big
    Write-Output "wrote tray-$Name.png"
}

Render-State -Name "playing" -Alpha 1.0
Render-State -Name "paused" -Alpha 0.45

# A preview of both, side by side, on a dark surface.
$bg = New-Object System.Drawing.Bitmap 320, 160
$g = [System.Drawing.Graphics]::FromImage($bg)
$g.Clear([System.Drawing.Color]::FromArgb(255, 40, 40, 40))
$g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor
$p1 = [System.Drawing.Image]::FromFile("Y:\Learn\Go\youtube-playlist\src\assets\tray-playing.png")
$p2 = [System.Drawing.Image]::FromFile("Y:\Learn\Go\youtube-playlist\src\assets\tray-paused.png")
$g.DrawImage($p1, 20, 20, 120, 120)
$g.DrawImage($p2, 180, 20, 120, 120)
$g.Dispose(); $p1.Dispose(); $p2.Dispose()
$bg.Save("C:\Users\yafit\AppData\Local\Temp\opencode\tray-states.png", [System.Drawing.Imaging.ImageFormat]::Png)
$bg.Dispose()
Write-Output "wrote preview"
