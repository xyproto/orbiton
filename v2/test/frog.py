#!/usr/bin/env python
# -*-coding:utf8-*-

import os
import time
import sys
import pygame
import subprocess

# this was standard for pygame back in the day
from pygame.locals import *

try:
    import psyco
    psyco.full()
except ImportError:
    pass


def a():
    """Test comment 1"""
    print("a")


def b():
    """
    Test comment 2
    """
    print("b")


class List:

    def __init__(self, screen, top, left, width, height, itemlist, selectednr, fg, bg, fontsize=32, fontface="superfrog.ttf"):
        self.screen = screen
        self.top = top
        self.left = left
        self.width = width
        self.height = height
        self.set(itemlist, selectednr)
        self.fg = fg
        self.bg = bg
        self.linewidth = 2
        self.margin = 10
        self.fontface = fontface
        self.fontsize = fontsize
        self.font = pygame.font.Font(self.fontface, self.fontsize)
        self.fontheight = self.font.size("Blabla")[1]
        self.rows = self.height / self.fontheight
        self.drawwidth = self.width - (self.linewidth * 2) - (self.margin * 2)
        self.selected_color = (255, 40, 20)
        self.clear_read_ratio_for_long_text = 0.7
        self.too_long_text_color = (80, 80, 75)

    def set(self, itemlist, selectednr):
        self.itemlist = itemlist
        self.selectednr = selectednr
        self.listlength = len(self.itemlist)
        if self.listlength == 0:
            print("Can't draw an empty list!")
            sys.exit(1)

    def draw(self):
        self.clear()

        gfx_rows = self.rows
        if len(self.itemlist) <= gfx_rows:
            # print "case 1: short list"
            startrow = 0
            endrow = self.listlength
        elif self.selectednr <= int(gfx_rows / 2):
            # print "case 2: selected at the top"
            startrow = 0
            endrow = startrow + gfx_rows
        elif self.selectednr > (self.listlength - int(gfx_rows / 2) - 1):
            # print "case 3: selected at the bottom"
            startrow = self.selectednr - int(gfx_rows / 2)
            endrow = len(self.itemlist)
        else:
            # print "case 4: selected in a long list"
            startrow = self.selectednr - int(gfx_rows / 2)
            endrow = startrow + gfx_rows

        selected_surface = None
        selected_pos = None
        for list_index in range(startrow, endrow):
            # Draw the text in red if it's selected, self.fg otherwise
            if list_index == self.selectednr:
                fg = self.selected_color
            else:
                fg = self.fg

            # Generate the graphics for for the list_index we're at
            gfx_index = list_index - startrow
            y = self.linewidth + self.top + gfx_index * self.fontheight
            text = self.font.render(self.itemlist[list_index], True, fg, self.bg)

            # If the text is too wide, use the left part of the image and scale down the rest
            if text.get_width() > self.drawwidth:
                leftsize = int(self.drawwidth * self.clear_read_ratio_for_long_text)

                # Try rendering until we have enough characters for the left part
                s = self.itemlist[list_index]
                numchars = len(s)
                lefttext = text
                while lefttext.get_width() > leftsize:
                    numchars -= 1
                    lefttext = self.font.render(s[:numchars-1], True, fg, self.bg)
                leftsize = lefttext.get_width()
                rightsize = self.drawwidth - leftsize

                # Now we have the left part in lefttext
                # Render the right part, and scale it down

                righttext = self.font.render(s[numchars-1:], True, self.too_long_text_color, self.bg)
                righttext = pygame.transform.scale(righttext, (rightsize, text.get_height()))

                # Now, blit the left and right part into a blank text-surface
                text.fill(self.bg)
                text = pygame.transform.scale(text, (leftsize+rightsize, text.get_height()))
                text.blit(lefttext, (0, 0))
                text.blit(righttext, (leftsize, 0))

            # Save the graphics if we're at the selected item, and draw the graphics
            if list_index == self.selectednr:
                selected_surface = text
                selected_pos = (self.left + self.linewidth + self.margin, y)
                self.screen.blit(text, selected_pos)
            else:
                self.screen.blit(text, (self.left + self.linewidth + self.margin, y))

        # Return the surface of the selected item, together with the rectangle
        # This is useful if some other function wishes to animate the text
        return selected_surface, selected_pos

    def __next__(self):
        self.selectednr += 1
        if self.selectednr == self.listlength:
            self.selectednr = 0

    def prev(self):
        self.selectednr -= 1
        if self.selectednr == -1:
            self.selectednr = self.listlength - 1

    def jumptoletter(self, letter):
        """Jump to the next item in the list starting with a certain letter. Returns True if we actually moved."""

        # items that starts with that letter
        itemnumbers = []
        for i, item in enumerate(self.itemlist):
            if item[0] == letter:
                itemnumbers.append(i)

        if itemnumbers:
            # If we're past the last item of that letter, or before the first one...
            if (self.selectednr > itemnumbers[-1]) or (self.selectednr < itemnumbers[0]):
                # go to the first number of that letter
                self.selectednr = itemnumbers[0]
                return True
            else:
                # go to the next instance of that letter
                for i in itemnumbers:
                    if i > self.selectednr:
                        self.selectednr = i
                        return True
                self.selectednr = itemnumbers[0]
                return True
        else:
            # Select the last letter on the list that is
            # still earlier in the alphabet (or else 0)
            chosen = 0
            for i, item in enumerate(self.itemlist):
                if ord(item[0]) < ord(letter):
                    chosen = i
            if self.selectednr == chosen:
                return False
            else:
                self.selectednr = chosen
                return True

    def selected(self):
        return self.itemlist[self.selectednr]

    def pageup(self):
        numup = min(self.rows / 2, self.selectednr)
        for x in range(numup):
            self.prev()

    def pagedown(self):
        numdown = min(self.rows / 2, self.listlength - self.selectednr - 1)
        for x in range(numdown):
            next(self)

    def home(self):
        self.selectednr = 0

    def end(self):
        self.selectednr = self.listlength - 1

    def clear(self):
        pygame.draw.rect(self.screen, self.bg, (self.left, self.top, self.width, self.height), 0)
        pygame.draw.rect(self.screen, self.fg, (self.left, self.top, self.width, self.height), self.linewidth)


class MenuProgram:

    def __init__(self, width, height, bg, fg, layout=[0.2, 0.2, 0.6], splashtime=0.9, fullscreen=False):
        self.width = width
        self.height = height
        self.bg = bg
        self.fg = fg

        self.menu = Menu()
        self.layout = layout
        if sum(layout) != 1:
            print("layout must be 1 in total!")
            sys.exit(1)

        pygame.display.init()
        pygame.font.init()

        if fullscreen:
            self.screen = pygame.display.set_mode((self.width, self.height), FULLSCREEN)
        else:
            self.screen = pygame.display.set_mode((self.width, self.height))

        pygame.display.set_caption("Superfrog")
        pygame.mouse.set_visible(1)

        self.clock = pygame.time.Clock()

        # --- Splash screen ---
        self.splashimage = None
        self.splash()
        pygame.display.flip()
        time.sleep(splashtime)

        # --- Menu screen ---
        # self.screen.fill(self.bg)

        self.lwidth = int(self.width * layout[0]) + 10
        self.l = List(self.screen, top=10, left=10, width=self.lwidth, height=self.height -
                      20, itemlist=self.menu.lmenu(), selectednr=0, fg=self.fg, bg=self.bg)
        self.l.draw()

        self.rwidth = int(self.width * layout[1]) + 10
        skew = 30
        self.r = List(self.screen, top=10+skew, left=10 + self.rwidth, width=self.rwidth,
                      height=self.height - 20 - skew, itemlist=self.menu.rmenu(), selectednr=0, fg=self.fg, bg=self.bg)
        # self.r.clear()

        self.active = self.l

        pygame.display.flip()

        # --- Mainloop ---
        self.wait_answer()

    def splash(self):
        if self.splashimage:
            scaled_image = self.splashimage
        else:
            image = pygame.image.load("superfrog.png")
            scaled_image = pygame.transform.scale(image, (self.width, self.height))
        return self.screen.blit(scaled_image, (0, 0))

    def on_select(self, text):
        if self.active == self.l:
            self.active.clear()
            self.menu.lselect(text)

            self.r.set(self.menu.rmenu(), 0)
            # self.r = List(self.screen, top=10, left=10 + self.rwidth, width=self.rwidth, height=self.height - 20, itemlist=self.menu.rmenu(), selectednr=0, fg=self.fg, bg=self.bg)

            self.active = self.r
        else:
            self.menu.execute(text)

    def on_back(self):
        if self.active == self.r:
            self.splash()
            self.menu.back()
            self.active = self.l
            self.active.draw()
        # else:
        #    print "can't back from left"

    def on_move(self):
        if self.active == self.r:
            # print "display content about", self.active.selected()
            pass

    def wait_answer(self):
        LETTERS = list(map(ord, "abcdefghijklmnopqrstuvwxyz" + chr(230) + chr(248) + chr(229)))

        keep_running = True

        # --- Mainloop ---
        while keep_running:

            for event in pygame.event.get():
                if event.type == QUIT:
                    keep_running = False
                elif event.type == KEYDOWN:
                    if event.key in [K_ESCAPE, K_LEFT]:
                        if self.active == self.l:
                            keep_running = False
                        else:
                            self.on_back()
                    elif event.key == K_DOWN:
                        next(self.active)
                        self.on_move()
                        self.active.draw()
                    elif event.key == K_UP:
                        self.active.prev()
                        self.on_move()
                        self.active.draw()
                    elif event.key in [K_RETURN, K_RIGHT]:
                        self.on_select(self.active.selected())
                        self.on_move()
                        self.active.draw()
                    elif event.key == K_PAGEUP:
                        self.active.pageup()
                        self.on_move()
                        self.active.draw()
                    elif event.key in [K_PAGEDOWN, K_SPACE]:
                        self.active.pagedown()
                        self.on_move()
                        self.active.draw()
                    elif event.key == K_HOME:
                        self.active.home()
                        self.on_move()
                        self.active.draw()
                    elif event.key == K_END:
                        self.active.end()
                        self.on_move()
                        self.active.draw()
                    elif event.key:
                        if event.key in LETTERS:
                            if self.active.jumptoletter(chr(event.key)):
                                self.on_move()
                                self.active.draw()
                        # else:
                        #    print "bah", event.key
                    else:
                        pass

            pygame.display.flip()

            # self.clock.tick(60)


def popen3(cmd, mode='t', bufsize=-1):
    p = subprocess.Popen(cmd, shell=True, bufsize=bufsize, stdin=PIPE, stdout=PIPE, stderr=PIPE, close_fds=True)
    return p.stdin, p.stdout, p.stderr


class Menu:

    def __init__(self, lselected=0, rselected=0):
        self.lactive = True
        self.ractive = False
        self.lselected = lselected
        self.rselected = rselected
        self.menudata = open("menu.txt").read().replace("    ", "\t")

        self.commands = {}

        current_category = ""
        current_name = ""
        current_command = ""

        self.loptions = {0: []}
        self.roptions = {}

        for line in self.menudata.strip().split("\n"):
            if not line:
                continue
            if (not line.startswith(" ")) and (not line.startswith("\t")):
                current_category = line.strip()
                self.loptions[0].append(current_category)
            elif line.strip().startswith("*"):
                current_name = line.strip()[1:].strip()
                if self.loptions[0].index(current_category) not in self.roptions:
                    self.roptions[self.loptions[0].index(current_category)] = [current_name]
                else:
                    self.roptions[self.loptions[0].index(current_category)].append(current_name)
            elif line.strip().startswith("!"):
                current_command = line.strip()[1:].strip()
                self.roptions[self.loptions[0].index(current_category)] = popen3(
                    current_command)[1].read().split("\n")[:-1]
            else:
                current_command = line.strip()
                self.commands[current_name] = current_command

    def execute(self, what):
        if what in self.commands:
            os.system(self.commands[what])
        else:
            print("No command for:", what)
            print(self.commands)

    def lselect(self, text):
        if self.lactive:
            self.lselected = self.loptions[0].index(text)
            self.lactive = False
            self.ractive = True

    def back(self):
        self.lactive = True
        self.ractive = False

    def lmenu(self):
        return self.loptions[0]

    def rmenu(self):
        return self.roptions[self.lselected]


def main():

    W = 1024
    H = 768
    BG = (255, 255, 220)
    FG = (5, 5, 10)
    LAYOUT = [0.2, 0.2, 0.6]
    SPLASH = 0.5
    FULL = False

    mp = MenuProgram(W, H, BG, FG, LAYOUT, SPLASH, FULL)

    pygame.display.quit()
    print("Bye!")
    sys.exit(0)


if __name__ == "__main__":
    main()
