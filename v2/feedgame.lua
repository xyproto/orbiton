canvas_width = 80
canvas_height = 24

local bobRuneLarge = 'O'
local bobRuneSmall = 'o'
local evilGobblerRune = '€'
local bubbleRune = '°'
local gobblerRune = 'G'
local gobblerDeadRune = 'T'
local gobblerZombieRune = '@'
local bobWonRune = 'Y'
local bobLostRune = 'n'
local pelletRune = '¤'

local bobColor = "lightyellow"
local bobWonColor = "lightgreen"
local bobLostColor = "red"
local evilGobblerColor = "lightred"
local gobblerColor = "yellow"
local gobblerDeadColor = "darkgray"
local gobblerZombieColor = "lightblue"
local bubbleColor = "magenta"
local pelletColor1 = "lightgreen"
local pelletColor2 = "green"
local statusTextColor = "black"
local statusTextBackground = "blue"
local resizeColor = "lightmagenta"
local gameBackgroundColor = "defaultbackground"

local bob = {}
local evilGobbler = {}
local gobblers = {}
local pellets = {}
local bubbles = {}
local highScore = 0
local score = 0
local running = true
local paused = false

function initGame(width, height)
    bob = { x = math.floor(width / 20), y = 10, oldx = math.floor(width / 20), oldy = 10, state = bobRuneSmall, color = bobColor, w = width, h = height }
    evilGobbler = { x = math.floor(width / 2) + 5, y = 10, oldx = math.floor(width / 2) + 5, oldy = 10, state = evilGobblerRune, color = evilGobblerColor, shot = false, w = width, h = height }
    gobblers = {}
    for i = 1, 25 do
        table.insert(gobblers, { x = math.floor(width / 2), y = 10, oldx = math.floor(width / 2), oldy = 10, state = gobblerRune, color = gobblerColor, dead = false, w = width, h = height })
    end
    bubbles = {}
    for i = 1, 15 do
        table.insert(bubbles, { x = math.floor(width / 5), y = 10, oldx = math.floor(width / 5), oldy = 10, state = bubbleRune, color = bubbleColor, w = width, h = height })
    end
    pellets = {}
    highScore = loadHighScore()
    score = 0
    running = true
    paused = false
end

function resizeGame(width, height)
    bob.color = resizeColor
    bob.w = width
    bob.h = height
    for _, gobbler in ipairs(gobblers) do
        gobbler.color = resizeColor
        gobbler.w = width
        gobbler.h = height
    end
    for _, bubble in ipairs(bubbles) do
        bubble.color = resizeColor
        bubble.w = width
        bubble.h = height
    end
    evilGobbler.color = resizeColor
    evilGobbler.w = width
    evilGobbler.h = height
end

function updateGame(key)
    local moved = false
    if key == 253 or key == 119 then  -- Up or w
        bob.y = bob.y - 1
        moved = true
    elseif key == 255 or key == 115 then  -- Down or s
        bob.y = bob.y + 1
        moved = true
    elseif key == 254 or key == 100 then  -- Right or d
        bob.x = bob.x + 1
        moved = true
    elseif key == 252 or key == 97 then  -- Left or a
        bob.x = bob.x - 1
        moved = true
    elseif key == 32 then  -- Space
        table.insert(pellets, { x = bob.x, y = bob.y, oldx = bob.x, oldy = bob.y, vx = bob.x - bob.oldx, vy = bob.y - bob.oldy, state = pelletRune, color = pelletColor1 })
    elseif key == 17 then  -- ctrl-q
        running = false
    elseif key == 114 then  -- r
        initGame(bob.w, bob.h)
    end

    if moved then
        if bob.state == bobRuneLarge then
            bob.state = bobRuneSmall
        else
            bob.state = bobRuneLarge
        end
    end

    for _, pellet in ipairs(pellets) do
        pellet.oldx = pellet.x
        pellet.oldy = pellet.y
        pellet.x = pellet.x + pellet.vx
        pellet.y = pellet.y + pellet.vy
        if pellet.x < 0 or pellet.x >= bob.w or pellet.y < 0 or pellet.y >= bob.h then
            pellet.removed = true
        end
    end

    for i = #pellets, 1, -1 do
        if pellets[i].removed then
            table.remove(pellets, i)
        end
    end

    for _, gobbler in ipairs(gobblers) do
        if not gobbler.dead then
            gobbler.oldx = gobbler.x
            gobbler.oldy = gobbler.y
            moveGobbler(gobbler)
        end
    end

    if not evilGobbler.shot then
        evilGobbler.oldx = evilGobbler.x
        evilGobbler.oldy = evilGobbler.y
        moveEvilGobbler(evilGobbler)
    end

    if not paused then
        for _, bubble in ipairs(bubbles) do
            bubble.oldx = bubble.x
            bubble.oldy = bubble.y
            moveBubble(bubble)
        end
    end

    checkCollisions()
end

function drawGame()
    plot(bob.x, bob.y, bob.color, bob.state)

    for _, pellet in ipairs(pellets) do
        plot(pellet.x, pellet.y, pellet.color, pellet.state)
    end

    plot(evilGobbler.x, evilGobbler.y, evilGobbler.color, evilGobbler.state)

    for _, gobbler in ipairs(gobblers) do
        plot(gobbler.x, gobbler.y, gobbler.color, gobbler.state)
    end

    for _, bubble in ipairs(bubbles) do
        plot(bubble.x, bubble.y, bubble.color, bubble.state)
    end

    local statusLine = "Score: " .. tostring(score) .. " | High Score: " .. tostring(highScore)
    write(0, 0, statusTextColor, statusTextBackground, statusLine)
end

function moveGobbler(gobbler)
    if gobbler.hunting == nil or gobbler.hunting.removed then
        local minDistance = math.huge
        for _, pellet in ipairs(pellets) do
            if not pellet.removed then
                local distance = math.sqrt((pellet.x - gobbler.x)^2 + (pellet.y - gobbler.y)^2)
                if distance < minDistance then
                    minDistance = distance
                    gobbler.hunting = pellet
                end
            end
        end
    end

    if gobbler.hunting then
        if gobbler.x < gobbler.hunting.x then
            gobbler.x = gobbler.x + 1
        elseif gobbler.x > gobbler.hunting.x then
            gobbler.x = gobbler.x - 1
        end
        if gobbler.y < gobbler.hunting.y then
            gobbler.y = gobbler.y + 1
        elseif gobbler.y > gobbler.hunting.y then
            gobbler.y = gobbler.y - 1
        end

        if gobbler.x == gobbler.hunting.x and gobbler.y == gobbler.hunting.y then
            gobbler.hunting.removed = true
            gobbler.hunting = nil
            score = score + 1
        end
    end
end

function moveEvilGobbler(evilGobbler)
    local minDistance = math.huge
    for _, gobbler in ipairs(gobblers) do
        if not gobbler.dead then
            local distance = math.sqrt((gobbler.x - evilGobbler.x)^2 + (gobbler.y - evilGobbler.y)^2)
            if distance < minDistance then
                minDistance = distance
                evilGobbler.hunting = gobbler
            end
        end
    end

    if evilGobbler.hunting then
        if evilGobbler.x < evilGobbler.hunting.x then
            evilGobbler.x = evilGobbler.x + 1
        elseif evilGobbler.x > evilGobbler.hunting.x then
            evilGobbler.x = evilGobbler.x - 1
        end
        if evilGobbler.y < evilGobbler.hunting.y then
            evilGobbler.y = evilGobbler.y + 1
        elseif evilGobbler.y > evilGobbler.hunting.y then
            evilGobbler.y = evilGobbler.y - 1
        end

        if evilGobbler.x == evilGobbler.hunting.x and evilGobbler.y == evilGobbler.hunting.y then
            evilGobbler.hunting.dead = true
            evilGobbler.hunting.state = gobblerDeadRune
            evilGobbler.hunting.color = gobblerDeadColor
            evilGobbler.hunting = nil
        end
    end
end

function moveBubble(bubble)
    bubble.x = bubble.x + math.random(-1, 1)
    bubble.y = bubble.y + math.random(-1, 1)

    if bubble.x < 0 then bubble.x = 0 end
    if bubble.x >= bubble.w then bubble.x = bubble.w - 1 end
    if bubble.y < 0 then bubble.y = 0 end
    if bubble.y >= bubble.h then bubble.y = bubble.h - 1 end
end

function checkCollisions()
    for _, gobbler in ipairs(gobblers) do
        if not gobbler.dead then
            for _, pellet in ipairs(pellets) do
                if not pellet.removed and gobbler.x == pellet.x and gobbler.y == pellet.y then
                    gobbler.dead = true
                    gobbler.state = gobblerDeadRune
                    gobbler.color = gobblerDeadColor
                    pellet.removed = true
                end
            end
        end
    end

    for _, pellet in ipairs(pellets) do
        if evilGobbler.x == pellet.x and evilGobbler.y == pellet.y then
            evilGobbler.shot = true
            pellet.removed = true
        end
    end

    local gobblersAlive = 0
    for _, gobbler in ipairs(gobblers) do
        if not gobbler.dead then
            gobblersAlive = gobblersAlive + 1
        end
    end

    if gobblersAlive == 0 then
        paused = true
        bob.state = bobLostRune
        bob.color = bobLostColor
        statusTextBackground = bobLostColor
        if score > highScore then
            saveHighScore(score)
        end
    elseif evilGobbler.shot then
        paused = true
        bob.state = bobWonRune
        bob.color = bobWonColor
        statusTextBackground = bobWonColor
        if score > highScore then
            saveHighScore(score)
        end
    end
end
