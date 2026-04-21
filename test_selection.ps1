$baseUrl = "http://localhost:8080/api/v1"
$lockerId = 123

function Get-LockerCounts {
    $resp = Invoke-RestMethod -Uri "$baseUrl/lockers" -Method Get
    $locker = $resp.data | Where-Object { $_.id -eq $lockerId }
    if (-not $locker) {
        Write-Error "Locker $lockerId not found in data"
        return $null
    }
    return $locker.freeCells
}

Write-Host "--- Step 1: Baseline ---"
$before = Get-LockerCounts
if (-not $before) { exit 1 }
Write-Host "Before: S:$($before.s), M:$($before.m), L:$($before.l), XL:$($before.xl)"

Write-Host "`n--- Step 2: Selection 1 (dimensions) ---"
$body1 = @{ dimensions = @{ length = 35; width = 30; height = 20; unit = "cm" } }
try {
    $sel1 = Invoke-RestMethod -Uri "$baseUrl/lockers/$lockerId/cell-selection" -Method Post -Body ($body1 | ConvertTo-Json) -ContentType "application/json"
    Write-Host "Selection 1 ID: $($sel1.data.selectionId), Size: $($sel1.data.size)"
} catch {
    Write-Host "Selection 1 failed: $_"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Host "Body: $($reader.ReadToEnd())"
    }
}

Write-Host "`n--- Step 3: Selection 2 (size 'm') ---"
$body2 = @{ size = "m" }
try {
    $sel2 = Invoke-RestMethod -Uri "$baseUrl/lockers/$lockerId/cell-selection" -Method Post -Body ($body2 | ConvertTo-Json) -ContentType "application/json"
    $selectionId2 = $sel2.data.selectionId
    Write-Host "Selection 2 ID: $selectionId2"
} catch {
    Write-Host "Selection 2 failed: $_"
    exit 1
}

Write-Host "`n--- Step 4: After Selection Check ---"
$afterSelection = Get-LockerCounts
Write-Host "After Selection: S:$($afterSelection.s), M:$($afterSelection.m), L:$($afterSelection.l), XL:$($afterSelection.xl)"

Write-Host "`n--- Step 5: Booking ---"
$body3 = @{ selectionId = $selectionId2; phone = "+79991112233" }
try {
    $booking = Invoke-RestMethod -Uri "$baseUrl/lockers/$lockerId/bookings" -Method Post -Body ($body3 | ConvertTo-Json) -ContentType "application/json"
    Write-Host "Booking ID: $($booking.data.id), Cell ID: $($booking.data.cellId)"
} catch {
    Write-Host "Booking failed: $_"
    if ($_.Exception.Response) {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        Write-Host "Body: $($reader.ReadToEnd())"
    }
    exit 1
}

Write-Host "`n--- Step 6: After Booking Check ---"
$afterBooking = Get-LockerCounts
Write-Host "After Booking: S:$($afterBooking.s), M:$($afterBooking.m), L:$($afterBooking.l), XL:$($afterBooking.xl)"

Write-Host "`n--- Step 7: Summary ---"
$results = @()
$results += [PSCustomObject]@{ Stage = "Before"; S = $before.s; M = $before.m; L = $before.l; XL = $before.xl }
$results += [PSCustomObject]@{ Stage = "After Selection"; S = $afterSelection.s; M = $afterSelection.m; L = $afterSelection.l; XL = $afterSelection.xl }
$results += [PSCustomObject]@{ Stage = "After Booking"; S = $afterBooking.s; M = $afterBooking.m; L = $afterBooking.l; XL = $afterBooking.xl }
$results | Format-Table

$sameSelection = ($before.s -eq $afterSelection.s -and $before.m -eq $afterSelection.m -and $before.l -eq $afterSelection.l -and $before.xl -eq $afterSelection.xl)
$correctBooking = ($afterBooking.m -eq ($before.m - 1))

Write-Host "Checks:"
Write-Host "- before == afterSelection ? $sameSelection"
Write-Host "- afterBooking.m == before.m - 1 ? $correctBooking"
