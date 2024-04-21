# Definition for singly-linked list.
# class ListNode(object):
#     def __init__(self, x):
#         self.val = x
#         self.next = None
class Solution(object):
    def getIntersectionNode(self, headA, headB):
        if headA == None or headB == None:
            return None
        
        hA,hB = headA, headB
        while hA != hB:
            hA = hA.next if hA else headB
            hB = hB.next if hB else headA
        return hA